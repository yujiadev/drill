package transport

import (
	"fmt"
	"log"	
	//"time"
	"net"
	//"sync"
	"crypto/rand"

	"drill/pkg/xcrypto"
)

type Server struct {
	Address string
	Pkey string		
	Protocal string
}

func NewServer(
	host string,
	port uint16,
	pkey, protocal string,
) Server {
	addr := fmt.Sprintf("%s:%v", host, port)	

	return Server {
		addr,
		pkey,
		protocal,
	}
}

func (txp *Server) Run() {
	udpAddr, err := net.ResolveUDPAddr("udp", txp.Address)
	if err != nil {
		log.Fatalf("Error resolving UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Error listening on UDP: %s", err)
	}

	buf := make([]byte, 65535)
	conns := make(map[uint64] chan []byte)
	counter := uint64(1)

	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP: %s", err)
			continue
		}

		cid, err := PeekConnectionId(&buf)
		if err != nil {
			log.Printf("Error peeking recv UDP packet: %s", err)
			continue
		}

		tx, ok := conns[cid]
		if ok {
			tx <- buf[0:n]
			continue
		}

		ch := make(chan []byte, 65535)
		new_cid := counter
		counter += 1
		conns[new_cid] = ch
		go tunnel(conn, addr, buf[0:n], new_cid, ch)
	}
}

func tunnel(
	conn *net.UDPConn,
	addr *net.UDPAddr, 
	data []byte, 
	cid uint64, 
	ch <-chan []byte,
) {
	cphr := xcrypto.NewXCipher("7abY7sBqNrtN5Z+NElo19hBDO1ixZ1+EGrrMq0gAjeE=")
	init, err := ParsePacket(&data, &cphr)
	if err != nil {
		log.Printf("Error parse INIT: %s", err)
		return
	}	
	fmt.Printf("Recv INIT (%v)\n", init.Method)

	// RETRY
	retry := NewRetry(cid, fmt.Sprintf("%s", addr), &cphr)
	conn.WriteToUDP(retry.Raw, addr)
	fmt.Println("Sent RETRY")

	// INIT2
	data = <-ch
	init2, err := ParsePacket(&data, &cphr)
	if err != nil {
		log.Printf("Error parse INIT2: %s", err)
		return
	}	
	fmt.Printf("Recv INIT2 (%v)\n", init2.Method)

	// INITACK
	ans := init2.Authenticate.Challenge
	id := init2.Authenticate.Id
	key := make([]byte, 32)
	rand.Read(key)

	initAck := NewInitAck(cid, id, ans, key, &cphr)
	conn.WriteToUDP(initAck.Raw, addr)

	// INTIDONE
	data = <-ch
	initDone, err := ParsePacket(&data, &cphr)
	if err != nil {
		log.Printf("Error parse INITDONE: %s", err)
		return
	}	
	fmt.Printf("Recv INITDONE (%v)\n", initDone.Method)

	// Negotiation is complete

	endpoints := make(map[uint64] chan Frame)
	for {
		cphrtxt := <- ch

		pkt, err := ParsePacket(&cphrtxt, &cphr)
		if err != nil {
			log.Printf("Server tunnel error: %s", err)
			continue
		}

		if pkt.Payload.Method == CONN {
			connectTask()
		}

		tx, ok := endpoints[pkt.Payload.Source]

		if !ok {
			log.Printf("Server can't find the endpoints")	
		}

		tx <- pkt.Payload
	}

}


func connectTask() {
	fmt.Println("Connect Task")
}

func sendTask() {

}

func recvTask() {

}




