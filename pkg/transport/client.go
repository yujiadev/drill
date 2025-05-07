package transport

import (
	"log"
	"fmt"
	"net"
	"sync"

	"drill/pkg/xcrypto"
)

type ClientTransport struct {
	laddr *net.UDPAddr
	raddr *net.UDPAddr
	id uint64
	pkey string
	protocal string
}

func NewClientTransport(
	host string,
	port uint16,
	remoteHost string,
	remotePort uint16,
	id uint64,
	pkey, protocal string,
) ClientTransport {
	host = fmt.Sprintf("%s:%v", host, port)
	remoteHost = fmt.Sprintf("%s:%v", remoteHost, remotePort)

	laddr, err := net.ResolveUDPAddr("udp", host)
	if err != nil {
		log.Fatalf("Err resolve UDP laddr %s\n", laddr)	
	}

	raddr, err := net.ResolveUDPAddr("udp", remoteHost)
	if err != nil {
		log.Fatalf("Err resolve UDP raddr %s\n", raddr)	
	}

	return ClientTransport {
		laddr,
		raddr,
		id,
		pkey,
		protocal,
	}
}

func (ct *ClientTransport) Run() {
	conn, cid, key := ct.clientHandshake()
	ch := make(chan Frame, 65535)
	chs := NewChannelMap[Frame]()
	var wg sync.WaitGroup
	wg.Add(1)

	go clientSendUDP(cid, 0, key, conn, ch)
	go clientRecvUDP(cid, 0, key, conn, chs)

	wg.Wait()
}

func (ct *ClientTransport) clientHandshake() (*net.UDPConn, uint64, []byte) {
	conn, err := net.DialUDP("udp", nil, ct.raddr)
	if err != nil {
		log.Fatalf("Err dail remote: %s\n", err)
	}

	cid := uint64(0)
	key := []byte{}
	buf := make([]byte, 65535)
	cphr := xcrypto.NewXCipher(ct.pkey)	

	//
	// INIT
	//
	init := NewInit(&cphr)
	if err := WriteAllUDP(conn, init.Raw); err != nil {
		log.Fatalf("Err send INIT: %s\n", err)		
	}
	fmt.Println("Sent INIT")

	//
	// RETRY
	//
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		log.Fatalf("Err recv RETRY: %s\n", err)
	}
	trimBuf := buf[:n]
	retry, err := ParsePacket(&trimBuf, &cphr)
	if err != nil {
		log.Fatalf("Err parse RETRY: %s\n", err)
	}
	fmt.Printf("Recv RETRY (%v)\n", retry.Method)

	//
	// INIT2
	//		
	cid = retry.ConnectionId
	token := retry.Token
	init2 := NewInit2(cid, ct.id, token, &cphr)
	if err := WriteAllUDP(conn, init2.Raw); err != nil {
		log.Fatalf("Err send INIT2: %s\n", err)		
	}
	fmt.Println("Sent INIT2")

	//
	// INITACK
	//
	n, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		log.Fatalf("Err recv INITACK: %s\n", err)
	}

	trimBuf = buf[:n]
	initAck, err := ParsePacket(&trimBuf, &cphr)
	key = initAck.Authenticate.Key
	if err != nil {
		log.Fatalf("Err parse INITACK: %s\n", err)
	}
	fmt.Printf("Recv INITACK (%v)\n", initAck.Method)

	// INITDONE
	ans := initAck.Authenticate.Challenge
	initDone := NewInitDone(cid, ct.id, ans, &cphr)
	if err := WriteAllUDP(conn, initDone.Raw); err != nil{
		log.Fatalf("Err send INITDONE: %s\n", err)		
	}
	fmt.Println("Sent INITDONE")

	return conn, cid, key
}

func clientSendUDP(
	cid, id uint64, 
	key []byte, 
	conn *net.UDPConn, 
	ch <-chan Frame,
) {
	cphr := xcrypto.NewXCipherFromBytes(key)

	for {
		frame := <-ch
		packet := NewTx(cid, id, frame, &cphr)

		if err := WriteAllUDP(conn, packet.Raw); err != nil {
			log.Fatalf("Err send packet (clientSendUDP): %s\n", err)
			continue
		}
	}
}

func clientRecvUDP(
	cid, id uint64, 
	key []byte, 
	conn *net.UDPConn, 
	chs *ChannelMap[Frame],
) {
	cphr := xcrypto.NewXCipherFromBytes(key)
	buf := make([]byte, 65535)

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("Err recv packet (clientRecvUDP): %s\n", err)
			continue
		}

		trimBuf := buf[:n]
		packet, err := ParsePacket(&trimBuf, &cphr)

		if err != nil {
			log.Fatalf("Err parse packet (clientRecvUDP): %s\n", err)
			continue
		}

		frame := packet.Payload
		if err := chs.Send(frame.Destination, &frame); err != nil {
			log.Fatalf("Err send frame (clientRecvUDP): %s\n", err)
			continue
		}
	}
}