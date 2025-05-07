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
	conns := NewChannelMap[[]byte]()

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

		ch, exists := conns.Get(cid)
		if !exists {
		 	ch, cid := conns.Create()
			go tunnel(conn, addr, buf[:n], cid, ch, conns)
			continue
		}
		ch <- buf[:n]
	}
}

func tunnel(
	conn *net.UDPConn,
	addr *net.UDPAddr, 
	data []byte, 
	cid uint64, 
	ch <-chan []byte,
	conns *ChannelMap[[]byte],
) {
	cphr := xcrypto.NewXCipher("7abY7sBqNrtN5Z+NElo19hBDO1ixZ1+EGrrMq0gAjeE=")
	init, err := ParsePacket(&data, &cphr)
	if err != nil {
		log.Printf("Error parse INIT: %s\n", err)
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
	endpoints := NewChannelMap[Frame]()

	for {
		cphrtxt := <- ch

		pkt, err := ParsePacket(&cphrtxt, &cphr)
		if err != nil {
			log.Printf("Server tunnel error: %s", err)
			continue
		}

		frame := pkt.Payload

		if frame.Method == CONN {
			go connectTask(frame, endpoints)
			continue
		}

		if err := endpoints.Send(frame.Destination, &frame); err != nil {
			log.Printf("Server can't find the endpoints")	
			continue
		}


	}
	
}


func sendToClient() {

}

func recvFromClient() {

}

func connectTask(frame Frame, endpoints *ChannelMap[Frame]) {
	host := string(frame.Payload)
	fmt.Printf("CONN request to %s\n", host)

	/*
	tcpConn, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("Err connect to %s: %s", host, err)
	}
	*/
}

func sendTask() {

}

func recvTask() {

}
