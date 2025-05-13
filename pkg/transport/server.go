package transport 

import (
	"log"
	//"time"
	"net"
	"sync"
)

type ExtFrame struct {
	Frame Frame
	Addr *net.UDPAddr
}

type ServerTransport struct {
	laddr		*net.UDPAddr	
	pkey		string
	protocol	string 
}

func NewServerTransport(
	laddr *net.UDPAddr,
	pkey, protocol string,
) ServerTransport {
	return ServerTransport {
		laddr,
		pkey,
		protocol,
	}
}

func (st *ServerTransport) Run() {
	conn, err := net.ListenUDP("udp", st.laddr)
	if err != nil {
		log.Fatalf("Error listen on UDP: %s\n", err)
	}

	sendCh := make(chan ExtFrame, 65535)

	var wg sync.WaitGroup
	wg.Add(2)
	go sendToClient(conn, sendCh, &wg)
	go recvFromClient(conn, sendCh, &wg)
	wg.Wait()
}

func sendToClient(
	conn *net.UDPConn, 
	sendCh <-chan ExtFrame, 
	wg *sync.WaitGroup,
) {
	for {
		extFrame := <-sendCh
		err := WriteAllUDPAddr(conn, extFrame.Addr, extFrame.Frame.Raw)

		if err != nil  {
			log.Printf("Err send frame to %s: %s\n", extFrame.Addr, err)
			continue
		}
	}

	wg.Done()
}

func recvFromClient(
	conn *net.UDPConn, 
	sendCh chan<-ExtFrame, 
	wg *sync.WaitGroup,
) {
	chMap := NewChannelMap[Frame]()
	buf := make([]byte, 65535)	

	for {
		n, raddr, err := conn.ReadFromUDP(buf)

		if err != nil {
			log.Printf("Err read UDP from client: %s\n", err)
			continue
		}

		//dataCopy := make([]byte, n)
		//copy(dataCopy, buf[:n])

		frame, err := ParseFrame(buf[:n])	

		if err != nil {
			log.Printf("Err parse UDP from client: %s\n", err)
			continue
		}

		if frame.Method == FCONN {
			recvCh, localID := chMap.Create()
			go delegateServer(sendCh, recvCh, raddr, frame, localID)
			continue
		}

		ch, exists := chMap.Get(frame.Dst)
		if !exists {
			log.Printf("Err route frame to dst: dst %v not exist\n", frame.Dst)
			continue
		}

		ch <- frame
		//frame.Report()
	}
}

func delegateServer(
	sendCh chan<-ExtFrame,
	recvCh <-chan Frame,
	raddr *net.UDPAddr,
	firstFrame Frame,
	localID uint64,
) {
	host := string(firstFrame.Payload)
	remoteID := firstFrame.Src

	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("Err connect to %s: %s\n", host, err)
		errFrame := NewFrame(FERR, 0, localID, remoteID, []byte("FERR"))
		extErrFrame := ExtFrame{ errFrame, raddr }
		sendCh <- extErrFrame
		return
	}

	log.Printf("Connection to %s established from server\n", host)

	okFrame := NewFrame(FOK, 0, localID, remoteID, []byte("FOK"))
	extOkFrame := ExtFrame{ okFrame, raddr }
	sendCh <- extOkFrame

	syncCh := make(chan Frame, 65535)

	var wg sync.WaitGroup
	wg.Add(2)	
	go delegateServerSend(
		conn,
		sendCh,
		syncCh,
		localID, 
		remoteID,
		raddr,
		&wg,
	)

	go delegateServerRecv(
		conn,
		recvCh,
		sendCh,
		syncCh,
		localID, 
		remoteID,
		raddr,
		&wg,
	)
	wg.Wait()
}

func delegateServerSend(
	conn net.Conn,
	sendCh chan<-ExtFrame,	
	syncCh <-chan Frame,
	localID, remoteID uint64,
	raddr *net.UDPAddr,
	wg *sync.WaitGroup,
) {
	pacer := NewSendPacer(localID, remoteID, 12)
	buf := make([]byte, 8196)	

	for {	
		log.Printf("(%v-%v) Sender waits more data being read\n", localID, remoteID)

		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Err read from target: %s\n", err)

			sendDone := NewSendDoneFrame(0, localID, remoteID)
			extSendDone := ExtFrame{ sendDone, raddr }	
			sendCh <- extSendDone
			log.Println("SENDDONE sent to client\n")

			wg.Done()
			return
		}

		//log.Printf("server side full bytes: %v\n", buf[:n])

		pacer.PushBuffer(buf[:n])
		readyFrames := pacer.GetReadyFrames()

		for _, frame := range readyFrames {
			extFrame := ExtFrame{ frame, raddr }
			sendCh <- extFrame
			log.Printf(
				"(%v-%v) Sender sent frame (%v), payload size: %v\n", 
				localID, 
				remoteID, 
				frame.Seq,
				len(frame.Payload),
			)
		}

		// Wait for Ack
		for {
			if len(pacer.GetSelectiveRepeat()) == 0 {
				break
			}

			log.Printf(
				"(%v-%v) Sender waits for more ACKS (%v more)\n", 
				localID,
				remoteID,
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
						log.Printf("(%v-%v) Sender recv RECVDONE\n", localID, remoteID)
						wg.Done()
						return
					}	
				/*
				case <-time.After(2*time.Second):
					repeatFrames := pacer.GetSelectiveRepeat()

					// Timeout resend the frames
					for _, frame := range repeatFrames {
						extFrame := ExtFrame{ frame, raddr }
						sendCh <- extFrame	
					}
				*/
			}
		}

		log.Printf("(%v-%v) Sender read more from target\n", localID, remoteID)
	}

	wg.Done()
} 

func delegateServerRecv(
	conn net.Conn,
	recvCh <-chan Frame,
	sendCh chan<-ExtFrame,	
	syncCh chan<-Frame,
	localID, remoteID uint64,
	raddr *net.UDPAddr,
	wg *sync.WaitGroup,
) {
	pacer := NewRecvPacer()

	for {
		log.Printf("(%v-%v) Receiver waits for more frames arriving", localID, remoteID)
		recvFrame := <-recvCh

		if recvFrame.Method == FACK || recvFrame.Method == FRECVDONE {
			syncCh <- recvFrame
			log.Printf(
				"(%v-%v) Receiver sync ACK (%v) with Sender\n", 
				localID, 
				remoteID,
				recvFrame.Seq, 
			)
			continue
		}

		if recvFrame.Method == FSENDDONE {
			log.Printf("(%v-%v) Receiver recv SENDDONE\n", localID, remoteID)
			wg.Done()
			return	
		}

		ackFrame := NewAckFrame(recvFrame.Seq, localID, remoteID)		
		extAckFrame := ExtFrame{ ackFrame, raddr }
		sendCh <- extAckFrame

		log.Printf("(%v-%v) Receiver sent ACK (%v)\n", localID, remoteID, ackFrame.Seq)
		pacer.PushFrame(recvFrame)
		readyFrames := pacer.GetReadyFrames()	

		//log.Printf("length of readyFrames: %v\n", len(readyFrames))
		//log.Printf("recvFrame payload: %v\n", recvFrame.Payload)

		for _, frame := range readyFrames {

			if frame.Method != FFWD {
				log.Printf("Should not happen! frame method: %v\n", frame.Method)
			}

			log.Printf(
				"(%v-%v) Receiver will send frame %v, %v bytes to target\n", 
				localID,
				remoteID,
				frame.Seq,
				len(frame.Payload),
			)
			log.Printf("server side target recv: %v\n", frame.Payload)

			if err := WriteAllTCP(conn, frame.Payload); err != nil {
				log.Printf("Err write to target: %s\n", err)

				recvDone := NewRecvDoneFrame(0, localID, remoteID)
				extRecvDone := ExtFrame{ recvDone, raddr }
				sendCh <- extRecvDone
				log.Println("RECVDONE sent to server\n")

				wg.Done()
				return 
			}

			log.Printf("(%v-%v) Receiver sent data to target\n", localID, remoteID)
		}
	}

	wg.Done()
}