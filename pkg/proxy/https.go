package proxy

import (
	"log"
	"fmt"
	"bufio"
	"net"
	"net/http"
)

type HttpsProxy struct {
	Address string
}

func NewHttpsProxy(addr string) HttpsProxy {
	return HttpsProxy { addr }
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