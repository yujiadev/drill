package proxy

import (
	"log"
	"fmt"
	"net"
)

type HttpsProxy struct {
	Address string
}

func New(host string, port uint16) HttpsProxy {
	addr := fmt.Sprintf("%s:%v", host, port)	
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
	fmt.Println("Handle HTTP CONNECT Request")	
}