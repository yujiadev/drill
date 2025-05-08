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

func serverRecvUDP(conn *net.UDPConn, key string) {
	chs := NewChannelMap[[]byte]()
	buf := make([]byte, 65535)

	for {
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("Err read UDP (serverRecvUDP): %s\n", err)
			continue
		}

		trimBuf := buf[:n]
		cid, err := PeekConnectionId(&trimBuf)
		if err != nil {
			log.Fatalf("Err peek CID (serverRecvUDP): %s\n", err)
			continue
		}

		ch, exists := chs.Get(cid)
		if !exists {
			ch, cid := chs.Create()
			go serverTunnel(conn, ch, trimBuf, cid, raddr, chs, key)
			continue
		}

		ch <- trimBuf
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
		packet, err := ParsePacket(&data, &cphr)
		if err != nil {
			log.Printf("Err parse packet (serverTunnel): %s\n", err)
			continue
		}

		frame := packet.Payload
		if frame.Method == CONN {
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
	init, err := ParsePacket(&firstUDP, &cphr)
	if err != nil {
		log.Fatalf("Err parse INIT: %s\n", err)
	}
	fmt.Printf("Recv INIT (%v)\n", init.Method)

	//
	// RETRY
	//
	retry := NewRetry(cid, fmt.Sprintf("%s", raddr), &cphr)
	if err := WriteAllUDPAddr(conn, retry.Raw, raddr); err != nil {
		log.Fatalf("Err send RETRY: %s\n", err)
	}
	fmt.Println("Sent RETRY")

	//
	// INIT2
	//
	data := <-ch
	init2, err := ParsePacket(&data, &cphr)
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
	initDone, err := ParsePacket(&data, &cphr)
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

	// Try to connect to the target host
	targetConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("Err connect target (connTarget): %s\n", err)
		errframe := NewFrame(ERR, 0, 0, dst, []byte("err"))
		errPacket := NewTx(cid, 0, errframe, &cphr)
		WriteAllUDPAddr(conn, errPacket.Raw, raddr)
		return
	}

	log.Printf("Connection to %s established\n", host)

	// Register an endpoints for subsequent forwarding
	ch, src := endpoints.Create()

	// Notify the client side endpoint
	okFrame := NewFrame(OK, 0, dst, src, []byte("ok"))
	okPacket := NewTx(cid, 0, okFrame, &cphr)
	WriteAllUDPAddr(conn, okPacket.Raw, raddr)

	// Spin up 2 goroutine to handle the fowarding
	ackCh := make(chan Frame, 65535)
	var wg sync.WaitGroup
	wg.Add(2)

	go sendTarget(targetConn, conn, ackCh, raddr, cid, src, dst, cphr, &wg)
	go recvTarget(targetConn, conn, ch, ackCh, raddr, cid, src, dst, cphr, &wg)

	wg.Wait()

	// Delete the endpoint
	endpoints.Delete(src)
}

// target send to client
func sendTarget(
	targetConn net.Conn, 
	conn *net.UDPConn,
	ackCh <-chan Frame,
	raddr *net.UDPAddr,
	cid, src, dst uint64,
	cphr xcrypto.XCipher,
	wg *sync.WaitGroup,
) {
	buf := make([]byte, 4096)
	seq := uint64(0)

	for {
		n, err := targetConn.Read(buf)

		if err != nil {
			finFrame := NewFrame(DISCONN, seq+1, src, dst, []byte("disconn"))
			packet := NewTx(cid, 0, finFrame, &cphr)

			if err := WriteAllUDPAddr(conn, packet.Raw, raddr); err != nil {
				log.Printf("Err send DISCONN to %s (serverTunnel): %s\n", raddr, err)
			}
			break
		}

		data := buf[:n]
		
		log.Printf("Written %v bytes\n", n)
		
		frame := NewFrame(FWD, seq, src, dst, data)
		packet := NewTx(cid, 0, frame, &cphr)

		if err := WriteAllUDPAddr(conn, packet.Raw, raddr); err != nil {
			log.Printf("Err send UDP to %s (serverTunnel): %s\n", raddr, err)
			break
		}

		for {
			ackFrame := <-ackCh

			if ackFrame.Sequence == seq {
				seq += 1
				break
			}

			if err := WriteAllUDPAddr(conn, packet.Raw, raddr); err != nil {
				log.Printf("Err send UDP to %s (serverTunnel): %s\n", raddr, err)
				break
			}
		}
	}

	wg.Done()
}

// target recv from client
func recvTarget(
	targetConn net.Conn, 
	conn *net.UDPConn,
	recvCh <-chan Frame, 
	ackCh chan<-Frame,
	raddr *net.UDPAddr,
	cid, src, dst uint64,
	cphr xcrypto.XCipher,
	wg *sync.WaitGroup,
) {
	for {
		frame := <-recvCh 
		fmt.Println("RECV")

		if frame.Method == ACK {
			ackCh <- frame
			continue
		}

		if frame.Method == FIN {
			break
		}

		written := 0
		for {
			n, err := targetConn.Write(frame.Payload[written:])
			if err != nil {
				wg.Done()
				return
			}

			written += n

			//log.Printf("Written %v bytes\n", written)

			if written == len(frame.Payload) {
				break
			}
		}

		// Send ACK to sender
		ackFrame := NewFrame(
			ACK, 
			frame.Sequence, 
			frame.Destination, 
			frame.Source, 
			[]byte("ack"),
		)

		packet := NewTx(cid, 0, ackFrame, &cphr)
		if err := WriteAllUDPAddr(conn, packet.Raw, raddr); err != nil {
			log.Printf("Err send ACK to %s (serverTunnel): %s\n", raddr, err)
		}

		fmt.Println("SENT")
	}

	wg.Done()
}
