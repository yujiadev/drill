package transport

import (
	"log"
	"fmt"
	"bufio"
	"net"
	"net/http"
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
	fmt.Printf("HttpsProxy is listening on %s\n", pxy.Address)

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