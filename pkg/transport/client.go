package transport 

import (
	"log"
	//"time"
	"net"
	"sync"
)

type ClientTransport struct {
	laddr 		*net.TCPAddr
	raddr 		*net.UDPAddr	
	pkey 		string
	protocol 	string
}

func NewClientTransport(
	laddr *net.TCPAddr,
	raddr *net.UDPAddr,
	pkey, protocol string,
) ClientTransport {
	return ClientTransport {
		laddr,
		raddr,
		pkey,
		protocol,
	}
}

func (ct *ClientTransport) Run() {
	sendCh := make(chan Frame, 65535)
	delegateCh := make(chan ConfirmedConn, 65535)
	chMap := NewChannelMap[Frame]()
	conn, err := net.DialUDP("udp", nil, ct.raddr)

	if err != nil {
		log.Fatalf("Err dial %s: %s\n", ct.raddr, err)
	}

	go sendToServer(conn, sendCh)
	go recvFromServer(conn, chMap)
	go delegateClientDispatch(delegateCh, sendCh, chMap) 

	// Start the HTTPS proxy
	https := NewHttpsProxy(
		ct.laddr,
		sendCh,
		delegateCh,
		chMap,
	)

	https.Run()
}

func sendToServer(conn *net.UDPConn, sendCh <-chan Frame) {
	for {
		frame := <-sendCh

		if err := WriteAllUDP(conn, frame.Raw); err != nil {
			log.Printf("Err send to server: %s\n", err)
			continue
		}
	}
}

func recvFromServer(conn *net.UDPConn, chMap *ChannelMap[Frame]) {
	buf := make([]byte, 65535)

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Err recv from server: %s\n", err)
			continue
		}

		frame, err := ParseFrame(buf[:n])
		if err != nil {
			log.Printf("Err parse recv frame from server: %s\n", err)
			continue
		}

		sendCh, exists := chMap.Get(frame.Dst)
		if !exists {
			log.Printf(
				"Err route frame from server: dst %v not exists\n",
				frame.Dst,
			)
			continue
		}

		sendCh <- frame
	}
}

func delegateClientDispatch(
	delegateCh <-chan ConfirmedConn, 
	sendCh chan<-Frame, 
	chMap *ChannelMap[Frame],
) {
	for {
		confirmed := <-delegateCh
		go delegateClient(confirmed, sendCh)
	}
}

func delegateClient(confirmed ConfirmedConn, sendCh chan<-Frame) {
	syncCh := make(chan Frame, 65535)
	var wg sync.WaitGroup
	wg.Add(2)

	go delegateClientSend(
		confirmed.Conn, 
		sendCh, 
		syncCh,
		confirmed.Src, 
		confirmed.Dst,
		&wg,
	)

	go delegateClientRecv(
		confirmed.Conn, 
		confirmed.RecvCh,
		sendCh, 
		syncCh,
		confirmed.Src, 
		confirmed.Dst,
		&wg,
	)

	wg.Wait()
}

func delegateClientSend(
	conn net.Conn, 
	sendCh chan<-Frame, 
	syncCh <-chan Frame,
	src, dst uint64,
	wg *sync.WaitGroup,
) {
	pacer := NewSendPacer(src, dst, 12)
	buf := make([]byte, 8196)	

	for {
		log.Printf("(%v-%v) Sender waiting for more data being read\n", src, dst)

		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Err read from client: %s\n", err)

			sendDone := NewSendDoneFrame(0, src, dst)
			sendCh <- sendDone
			log.Println("SENDDONE sent to server\n")

			wg.Done()
			return
		}

		pacer.PushBuffer(buf[:n])
		readyFrames := pacer.GetReadyFrames()

		for _, frame := range readyFrames {
			sendCh <- frame
			/*
			log.Printf(
				"(%v-%v) Sender sent frame (%v), payload size: %v\n", 
				src, 
				dst, 
				frame.Seq,
				len(frame.Payload),
			)
			log.Printf("sent frame data (%v): %v\n", len(frame.Payload), frame.Payload)
			*/
		}

		// Wait for Ack
		for {
			if len(pacer.GetSelectiveRepeat()) == 0 {
				break
			}

			log.Printf(
				"(%v-%v) Sender waits for more ACKS (%v more)\n", 
				src,
				dst,
				len(pacer.GetSelectiveRepeat()),
			)

			select {
				case syncFrame := <- syncCh:
					// Update pacer
					if syncFrame.Method == FACK {
						pacer.RecvAck(syncFrame.Seq)
						continue
					}

					// Server side no longer recv			
					if syncFrame.Method == FRECVDONE {
						log.Printf("(%v-%v) Sender recv RECVDONE\n", src, dst)
						wg.Done()
						return
					}	
				/*
				case <-time.After(2*time.Second):
					repeatFrames := pacer.GetSelectiveRepeat()

					// Timeout resend the frames
					for _, frame := range repeatFrames {
						sendCh <- frame
					}
				*/
			}
		}

		log.Printf("(%v-%v) Sender read more from client\n", src, dst)
	}

	wg.Done()
}

func delegateClientRecv(
	conn net.Conn, 
	recvCh <-chan Frame,
	sendCh chan<-Frame, 
	syncCh chan<-Frame,
	src, dst uint64,
	wg *sync.WaitGroup,
) {
	pacer := NewRecvPacer()

	for {
		log.Printf("(%v-%v) Receiver waits for more frames arriving", src, dst)
		recvFrame := <-recvCh

		if recvFrame.Method == FACK || recvFrame.Method == FRECVDONE {
			syncCh <- recvFrame
			log.Printf(
				"(%v-%v) Receiver sync ACK (%v) with Sender\n", 
				src, 
				dst,
				recvFrame.Seq,
			)
			continue
		}

		if recvFrame.Method == FSENDDONE {	
			log.Printf("(%v-%v) Receiver recv SENDDONE\n", src, dst)
			wg.Done()
			return	
		}

		ackFrame := NewAckFrame(recvFrame.Seq, src, dst)		
		sendCh <- ackFrame
		log.Printf("(%v-%v) Receiver sent ACK (%v)\n", src, dst, ackFrame.Seq)

		pacer.PushFrame(recvFrame)
		readyFrames := pacer.GetReadyFrames()	

		for _, frame := range readyFrames {

			if frame.Method != FFWD {
				log.Printf("Should not happen! frame method: %v\n", frame.Method)
			}	
			log.Printf(
				"(%v-%v) Receiver will send frame %v, %v bytes to client\n", 
				src,
				dst,
				frame.Seq,
				len(frame.Payload),
			)

			if err := WriteAllTCP(conn, frame.Payload); err != nil {
				log.Printf("Err write to client: %s\n", err)
				recvDone := NewRecvDoneFrame(0, src, dst)
				sendCh <- recvDone
				log.Println("RECVDONE sent to server\n")

				wg.Done()
				return 
			}

			log.Printf("(%v-%v) Receiver sent data to client\n", src, dst)
		}
	}

	wg.Done()
}