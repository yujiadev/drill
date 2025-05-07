package transport

import (
	"log"
	"fmt"
	"time"
	"bufio"
	"net"
	"net/http"

	"drill/pkg/xcrypto"
)

type HttpsProxy struct {
	Address string
	RemoteAddress string
	Pkey string
	Protocal string
}

func NewHttpsProxy(
	addr string,
	port uint16,
	raddr string,
	rport uint16,
	pkey, protocal string,
) HttpsProxy {
	address       := fmt.Sprintf("%s:%v", addr, port)
	remoteAddress := fmt.Sprintf("%s:%v", raddr, rport)

	return HttpsProxy { 
		address,
		remoteAddress,
		pkey,
		protocal,
	}
}

func (pxy *HttpsProxy) Run() {
	cphr := xcrypto.NewXCipher(pxy.Pkey)

	// Negotiate with remote server 	
	cid, _, udpConn := negotiate(pxy.RemoteAddress, pxy.Pkey)

	sendCh := make(chan Frame, 65535)
	go sendToServer(cid, 0, udpConn, sendCh, cphr)
	go recvFromServer(udpConn, cphr)

	// HTTPS proxy
	ln, err := net.Listen("tcp", pxy.Address)
	if err != nil {
		log.Fatalf("HttpsProxy::Run() error. %s", err)	
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Err accept HTTP request: %s\n", err)
			continue
		}

		go handleConnectRequest(conn, sendCh)
	}
}

func negotiate(raddr, pkey string) (uint64, []byte, *net.UDPConn) {
    time.Sleep(2 * time.Second)

    cid := uint64(0)
    id  := uint64(0)
    key := []byte{}
	buf := make([]byte, 65535)
    remoteAddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		log.Fatalf("Error resolve UDP address: %v\n", err)
	}

	cphr := xcrypto.NewXCipher(pkey)
	conn, _ := net.DialUDP("udp", nil, remoteAddr)

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

	bytes := buf[:n]
	retry, err := ParsePacket(&bytes, &cphr)
	if err != nil {
		log.Fatalf("Err parse RETRY: %s\n", err)
	}
	fmt.Printf("Recv RETRY (%v)\n", retry.Method)

	//
	// INIT2
	//
	cid = retry.ConnectionId
	token := retry.Token
	init2 := NewInit2(cid, id, token, &cphr)
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

	bytes = buf[:n]
	initAck, err := ParsePacket(&bytes, &cphr)
	key = initAck.Authenticate.Key
	if err != nil {
		log.Fatalf("Err parse INITACK: %s\n", err)
	}
	fmt.Printf("Recv INITACK (%v)\n", initAck.Method)

	// INITDONE
	ans := initAck.Authenticate.Challenge
	initDone := NewInitDone(cid, id, ans, &cphr)
	if err := WriteAllUDP(conn, initDone.Raw); err != nil{
		log.Fatalf("Err send INITDONE: %s\n", err)		
	}
	fmt.Println("Sent INITDONE")

	return cid, key, conn
}

func sendToServer(
	cid, id uint64,
	conn *net.UDPConn, 
	ch <-chan Frame, 
	cphr xcrypto.XCipher,
) {
	for {
		frame := <-ch
		packet := NewTx(cid, id, frame, &cphr)

		// Write all the bytes
		if err := WriteAllUDP(conn, packet.Raw); err != nil {
			log.Printf("Err send UDP: %s\n", err)
			continue
		}
	}
}

func recvFromServer(conn *net.UDPConn, cphr xcrypto.XCipher) {

}

func handleConnectRequest(conn net.Conn, sendCh chan<-Frame) {
	request, err := http.ReadRequest(bufio.NewReader(conn))

	if err != nil {
		log.Fatalf("Error read HTTP request: %s", err)
	}

	// Only support HTTP CONNECT method 
	if request.Method != "CONNECT" {
		log.Fatalf("Error not supported HTTP method")
	}

	connFrame := NewFrame(CONN, 0, 0, 0, []byte(string(request.Host)))
	sendCh <-connFrame	
}


