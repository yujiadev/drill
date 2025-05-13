package transport 

import (
	"log"
	"bufio"
	"net"
	"net/http"
)

type ConfirmedConn struct {
	Src uint64
	Dst uint64
	Conn net.Conn
	RecvCh <-chan Frame
	target string
}

type HttpsProxy struct {
	addr 		*net.TCPAddr
	sendCh 		chan<-Frame
	delegateCh 	chan<-ConfirmedConn
	chMap   	*ChannelMap[Frame]
}

func NewHttpsProxy(
	addr *net.TCPAddr, 
	sendCh chan<-Frame, 
	delegateCh 	chan<-ConfirmedConn,
	chMap *ChannelMap[Frame],
) HttpsProxy {
	return HttpsProxy {
		addr,
		sendCh,
		delegateCh,
		chMap,
	}
}

func (pxy *HttpsProxy) Run() {
	ln, err := net.ListenTCP("tcp", pxy.addr)	
	if err != nil {
		log.Fatalf("Err listen on %s: %s\n", pxy.addr, err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Err accept TCP: %s\n", err)
			continue
		}

		go handle(conn, pxy.sendCh, pxy.delegateCh, pxy.chMap)
	}
}

func handle(
	conn net.Conn,
	sendCh chan<-Frame, 
	delegateCh 	chan<-ConfirmedConn,
	chMap  *ChannelMap[Frame],
) {
	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Printf("Err read HTTP request: %s\n", err)
		return
	}

	// Only allow HTTP CONNECT method
	if req.Method != "CONNECT" {
		log.Println("Err HTTP method: only HTTP CONNECT method allowed")
		return
	}

	log.Println(req.Host)

	//
	// Negotiate
	//
	recvCh, src := chMap.Create()

	connFrame := NewConnFrame(src, req.Host)
	sendCh <- connFrame
	respFrame := <-recvCh

	// Remote server can't connect to target host, return
	if respFrame.Method != FOK {
		chMap.Delete(src)
		log.Printf("Err HTTP CONNECT request: can't fulfill\n")
		return
	}

	// Notify client tunnel is established
    _, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
    if err != nil {
		log.Printf("Err notify HTTPS client: %s\n", err)
		return
    }

    // Handle the subsequent tunneling
    confirmedConn := ConfirmedConn {
    	src,			// Local ID
    	respFrame.Src,	// Remote ID
    	conn,			// TCP connection the client "browswer"
    	recvCh,			
    	req.Host,		
    }

    delegateCh <- confirmedConn		// Let ClientTransport handle transfer
}