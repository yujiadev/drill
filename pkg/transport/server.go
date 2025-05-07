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
			go serverHandshake(conn, ch, trimBuf, cid, raddr, chs, key)
			continue
		}

		ch <- trimBuf
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
) {
	cphr := xcrypto.NewXCipher(pkey)

	//
	// INIT
	// 
	init, err := ParsePacket(&firstUDP, &cphr)
	if err != nil {
		log.Fatalf("Err parse INIT: %s\n", err)
		return
	}
	fmt.Printf("Recv INIT (%v)\n", init.Method)

	//
	// RETRY
	//
	retry := NewRetry(cid, fmt.Sprintf("%s", raddr), &cphr)
	if err := WriteAllUDPAddr(conn, retry.Raw, raddr); err != nil {
		log.Fatalf("Err send RETRY: %s\n", err)
		return
	}
	fmt.Println("Sent RETRY")

	//
	// INIT2
	//
	data := <-ch
	init2, err := ParsePacket(&data, &cphr)
	if err != nil {
		log.Printf("Error parse INIT2: %s\n", err)
		return
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
		return
	}
	fmt.Println("Sent INITACK")

	//
	// INITDONE	
	//
	data = <-ch
	initDone, err := ParsePacket(&data, &cphr)
	if err != nil {
		log.Printf("Error parse INITDONE: %s\n", err)
		return
	}	
	fmt.Printf("Recv INITDONE (%v)\n", initDone.Method)
}

