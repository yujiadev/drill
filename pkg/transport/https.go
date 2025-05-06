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

	negotiate(pxy.RemoteAddress, pxy.Pkey)

	ln, err := net.Listen("tcp", pxy.Address)
	if err != nil {
		log.Fatalf("HttpsProxy::Run() error. %s", err)	
	}

	for {
		conn, err := ln.Accept()
		if err != nil {

			continue
		}

		go handleConnectRequest(conn)
	}
}

func negotiate(raddr, pkey string) net.Conn {
    time.Sleep(3 * time.Second)

	cphr := xcrypto.NewXCipher(pkey)
	conn, _ := net.Dial("udp", raddr)
	buf := make([]byte, 65535)

	// INIT
	init := NewInit(&cphr)
	conn.Write(init.Raw)
	fmt.Println("Sent INIT")

	// RETRY
	buf = make([]byte, 65535)
	n, err := conn.Read(buf)

	if err != nil {
		fmt.Printf("Err: %s", err)
	}

	bytes := buf[:n]
	retry, err := ParsePacket(&bytes, &cphr)
	if err != nil {
		log.Fatalf("%s", err)
	}
	fmt.Printf("Recv RETRY (%v)\n", retry.Method)

	// INIT2
	cid := retry.ConnectionId
	id := retry.Id
	token := retry.Token
	init2 := NewInit2(cid, id, token, &cphr)
	conn.Write(init2.Raw)
	fmt.Println("Sent INIT2")

	// INITACK
	n, err = conn.Read(buf)
	if err != nil {
		fmt.Printf("Err: %s", err)
	}
	bytes = buf[:n]
	initAck, err := ParsePacket(&bytes, &cphr)

	if err != nil {
		log.Fatalf("%s", err)
	}
	fmt.Printf("Recv INITACK (%v)\n", initAck.Method)

	// INITDONE
	ans := initAck.Authenticate.Challenge
	initDone := NewInitDone(cid, id, ans, &cphr)

	conn.Write(initDone.Raw)
	fmt.Println("Sent INITDONE")

	return conn
}

func handleConnectRequest(conn net.Conn) {
	request, err := http.ReadRequest(bufio.NewReader(conn))

	if err != nil {
		log.Fatalf("handleConnectRequest read HTTP request error. %s", err)
	}

	// Only support HTTP CONNECT method 
	if request.Method != "CONNECT" {
		log.Fatalf("handleConnectRequest() error. Not supported HTTP method")
	}

	fmt.Printf("%s\n", request.Host)
	fmt.Printf("%s\n", conn.RemoteAddr())

	conn.Close()
}