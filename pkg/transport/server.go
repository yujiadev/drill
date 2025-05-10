package transport

import (
	"fmt"
	"log"
	"sync"
	"net"
	"crypto/rand"

	"drill/pkg/xcrypto"
)

type ServerTransport struct {
	laddr *net.UDPAddr
	pkey string
	protocal string
}

func NewServerTransport(
	host string, 
	port uint16, 
	pkey, protocal string,
) ServerTransport {
	host = fmt.Sprintf("%s:%v", host, port)
	laddr, err := net.ResolveUDPAddr("udp",  host)
	if err != nil {
		log.Fatalf("Err resolve UDP laddr %s\n", laddr)	
	}

	return ServerTransport {
		laddr,
		pkey,
		protocal,
	}
}

func (st *ServerTransport) Run() {	
	conn, err := net.ListenUDP("udp", st.laddr)
	if err != nil {
		log.Fatalf("Error listen on UDP: %s\n", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go serverRecvUDP(conn, st.pkey)
	wg.Wait()
}

func serverSendUDP(
	conn *net.UDPConn,
	ch <-chan Frame,
	cid uint64,
	raddr *net.UDPAddr, 
	cphr xcrypto.XCipher,
) {
	for {
		frame, ok := <-ch

		// Channel is closed
		if !ok {
			log.Println("Channel close (serverSendUDP)")
			return
		}

		packet := NewTx(cid, 0, frame, &cphr)
		err := WriteAllUDPAddr(conn, packet.Raw, raddr)

		if err != nil {
			log.Printf("Err send UDP (serverSendUDP): %s\n", err)
			continue
		}
	}
}

func serverRecvUDP(conn *net.UDPConn, key string) {
	chs := NewChannelMap[[]byte]()
	buf := make([]byte, 65535)

	for {
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("Err read UDP (serverRecvUDP): %s\n", err)
			continue
		}

		cid, err := PeekConnectionId(buf[:n])
		if err != nil {
			log.Fatalf("Err peek CID (serverRecvUDP): %s\n", err)
			continue
		}

		ch, exists := chs.Get(cid)
		if !exists {
			ch, cid := chs.Create()
			go serverTunnel(conn, ch, buf[:n], cid, raddr, chs, key)
			continue
		}

		ch <- buf[:n]
	}	
}

func serverTunnel(
	conn *net.UDPConn, 
	ch chan []byte, 
	firstUDP []byte,
	cid uint64, 
	raddr *net.UDPAddr,
	chs *ChannelMap[[]byte],
	pkey string,
) {
	cid, key := serverHandshake(conn, ch, firstUDP, cid, raddr, chs, pkey)
	cphr := xcrypto.NewXCipherFromBytes(key)
	endpoints := NewChannelMap[Frame]()

	for {
		data := <-ch
		packet, err := ParsePacket(data, &cphr)
		if err != nil {
			log.Printf("Err parse packet (serverTunnel): %s\n", err)
			continue
		}

		frame := packet.Payload
		if frame.Method == FCONN {
			go connTarget(conn, endpoints, cid, raddr, frame, cphr)
			continue
		}

		tx, exists := endpoints.Get(frame.Destination)
		if !exists {
			log.Printf(
				"Err channel frame (serverTunnel): dest (%v) not found\n", 
				frame.Destination,
			)
			continue
		}
		tx <- frame
	}		
}

func serverHandshake(
	conn *net.UDPConn, 
	ch chan []byte, 
	firstUDP []byte,
	cid uint64, 
	raddr *net.UDPAddr,
	chs *ChannelMap[[]byte],
	pkey string,
) (uint64, []byte) {
	cphr := xcrypto.NewXCipher(pkey)

	//
	// INIT
	// 
	init, err := ParsePacket(firstUDP, &cphr)
	if err != nil {
		log.Fatalf("Err parse INIT: %s\n", err)
	}
	fmt.Printf("Recv INIT (%v)\n", init.Method)

	//
	// RETRY
	//
	retry := NewRetry(cid, []byte(fmt.Sprintf("%s", raddr)), &cphr)
	if err := WriteAllUDPAddr(conn, retry.Raw, raddr); err != nil {
		log.Fatalf("Err send RETRY: %s\n", err)
	}
	fmt.Println("Sent RETRY")

	//
	// INIT2
	//
	data := <-ch
	init2, err := ParsePacket(data, &cphr)
	if err != nil {
		log.Printf("Error parse INIT2: %s\n", err)
	}	
	fmt.Printf("Recv INIT2 (%v)\n", init2.Method)

	//
	// INITACK
	//
	ans := init2.Authenticate.Challenge
	id := init2.Authenticate.Id
	key := make([]byte, 32)
	rand.Read(key)

	initAck := NewInitAck(cid, id, ans, key, &cphr)
	if err := WriteAllUDPAddr(conn, initAck.Raw, raddr); err != nil {
		log.Fatalf("Err send INITACK: %s\n", err)
	}
	fmt.Println("Sent INITACK")

	//
	// INITDONE	
	//
	data = <-ch
	initDone, err := ParsePacket(data, &cphr)
	if err != nil {
		log.Printf("Error parse INITDONE: %s\n", err)
	}	
	fmt.Printf("Recv INITDONE (%v)\n", initDone.Method)

	return cid, key
}

func connTarget(
	conn *net.UDPConn,
	endpoints *ChannelMap[Frame],
	cid uint64,
	raddr *net.UDPAddr,
	frame Frame,
	cphr xcrypto.XCipher,
) {
	host := string(frame.Payload)
	dst := frame.Source
	fmt.Println(host)

	// Try to connect to the target host
	targetConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("Err connect target (connTarget): %s\n", err)
		errframe := NewFrame(FERR, 0, 0, dst, []byte("err"))
		errPacket := NewTx(cid, 0, errframe, &cphr)
		WriteAllUDPAddr(conn, errPacket.Raw, raddr)
		return
	}

	log.Printf("Connection to %s established\n", host)

	// Register an endpoints for subsequent forwarding
	recvCh, src := endpoints.Create()

	// Notify the client side endpoint
	okFrame := NewFrame(FOK, 0, dst, src, []byte("ok"))
	okPacket := NewTx(cid, 0, okFrame, &cphr)
	WriteAllUDPAddr(conn, okPacket.Raw, raddr)

	// Spin up 3 goroutines to handle the fowarding
	notifyCh := make(chan Frame, 65535)
	sendCh := make(chan Frame, 65535)
	var wg sync.WaitGroup
	wg.Add(2)

	go serverSendUDP(conn, sendCh, cid, raddr, cphr)
	go sendTarget(targetConn, sendCh, notifyCh, src, dst, &wg)
	go recvTarget(targetConn, recvCh, sendCh, notifyCh, src, dst, &wg)

	wg.Wait()

	//close(sendCh)

	// Delete the endpoint
	endpoints.Delete(src)
}

// target send to client
func sendTarget(
	conn net.Conn, 
	ch chan<-Frame, 
	notifyCh <-chan Frame, 
	src, dst uint64,
	wg *sync.WaitGroup,
) {
	seq := uint64(0)

	for {
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)

		// target will no longer send any data
		if err != nil {
			sendDone := NewFrame(FSENDDONE, 0, src, dst, []byte("senddone"))
			ch <- sendDone
			break
		}
		
		frame := NewFrame(FFWD, seq, src, dst, buf[:n])
		ch <-frame

		respFrame := <-notifyCh

		// client will no longer recv any data
		if respFrame.Method == FRECVDONE {
			wg.Done()
			return
		}

		seq += 1
	}

	wg.Done()
}

// target recv from client
func recvTarget(
	conn net.Conn,
	recvCh <-chan Frame,	
	sendCh chan<-Frame,
	notifyCh chan<-Frame,
	src, dst uint64,
	wg *sync.WaitGroup,
) {

	for {
		frame := <-recvCh

		if frame.Method == FACK || frame.Method == FRECVDONE {
			notifyCh <- frame
			continue
		}

		if frame.Method == FSENDDONE {
			break
		}

		err := WriteAllTCP(conn, frame.Payload)
		if err != nil {
			log.Printf("Err send target: %s\n", err)
			break			
		}

		ackFrame := NewFrame(
			FACK,
			frame.Sequence,
			frame.Destination,
			frame.Source,
			[]byte("ack"),
		)

		sendCh <- ackFrame	
	}

	wg.Done()
}
