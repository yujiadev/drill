package transport

type Hold struct {}

/*
package transport

import (
	"log"
	"fmt"
	"time"
	"net"
	"sync"
	"crypto/rand"
	"drill/internal/obfuscate"
	"drill/pkg/netio"
)

type ClientTransport struct {
	laddr *net.TCPAddr
	raddr *net.UDPAddr
	pkey  []byte
	wg    *sync.WaitGroup
}

func NewClientTransport(
	laddr *net.TCPAddr,
	raddr *net.UDPAddr,
	pkey  []byte,
	wg    *sync.WaitGroup,
) ClientTransport {
	return ClientTransport {
		laddr,
		raddr,
		pkey,
		wg,
	}
}

func (ct *ClientTransport) Run() {
	conn, err := net.DialUDP("udp", nil, ct.raddr)

	if err != nil {
		log.Printf("Error dial %s: %s\n", ct.raddr, err)
		ct.wg.Done()
		return
	}

	sendCh := make(chan []byte, 65535)
	recvCh := make(chan []byte, 65535)
	go clientSocketRecv(conn, recvCh, ct.raddr)

	obfs := obfuscate.NewBasicObfuscator(ct.pkey)
	pkey2, cid, err := clientHandshake(conn, recvCh, obfs)
	if err != nil {
		log.Printf("Error handshake. %s\n", err)
		ct.wg.Done()
	}

	go clientHttpsProxy(conn, ct.laddr, cid, pkey2)

}

func clientSocketRecv(
	conn *net.UDPConn, 
	ch chan<-[]byte,
	addr *net.UDPAddr,
) {
	buf := make([]byte, 65535)
	for {
		n, raddr, err := conn.ReadFromUDP(buf)

		if err != nil {
			continue
		}

		if !raddr.IP.Equal(addr.IP) || raddr.Port != addr.Port {
			continue
		}

		data := make([]byte, 0, n)
		data = append(data, buf[:n]...)
		ch <- data
	}
}

func clientHttpsProxy(
	conn *net.UDPConn,  
	laddr *net.TCPAddr, 
	cid uint64,
	pkey2 []byte,
) {
	ln, err := net.ListenTCP("tcp", laddr)	
	if err != nil {
		log.Panicf("Err listen on %s: %s\n", laddr, err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Err accept TCP: %s\n", err)
			continue
		}

		go clientHandle(conn)
	}
}

func clientHandle(conn net.Conn) {
	host, err := ParseHTTPConnectHost(conn)
	if err != nil {
		log.Printf("Err parse HTTP CONNECT host: %s\n", host)
		return
	}

	log.Println(host)
}

func clientSend() {

}

func clientRecv() {

}

func clientHandshake(
	conn *net.UDPConn,  
	ch <-chan []byte,
	obfs obfuscate.Obfuscate,
) ([]byte, uint64, error) {
	pkey1 := make([]byte, 32)
	rand.Read(pkey1)

	if err := clientInit(conn, obfs, pkey1); err != nil {
		return []byte{}, 0, err
	}

	if err := clientRetry(conn, ch); err != nil {
		return []byte{}, 0, err
	}

	pkey2, cid, err := clientAuth(conn, ch, obfs, pkey1)
	if err != nil {
		return []byte{}, 0, err
	}

	return pkey2, cid, nil
}

func clientInit(
	conn *net.UDPConn, 
	obfs obfuscate.Obfuscate, 
	pkey1 []byte,
) error {
	pkt := NewInitPacket(pkey1)
	encoded := obfs.Encode(pkt.AsBytes())

	if err := netio.WriteUDP(conn, encoded); err != nil {
		return err
	}

	return nil
}

func clientRetry(conn *net.UDPConn, recvCh <-chan []byte) error {
	select {
	case data :=<-recvCh:
		if err := netio.WriteUDP(conn, data); err != nil {
			return err
		}

		return nil
	case <-time.After(2*time.Second):
		return fmt.Errorf("timeout on receving retry token from server")
	}
}

func clientAuth(
	conn *net.UDPConn, 
	recvCh <-chan []byte,
	obfs obfuscate.Obfuscate, 
	pkey1 []byte,
) ([]byte, uint64, error) {
	//
	// Recv AUTH packet from server, try to get the pkey2 from server
	//
	obfs.SetPkey(pkey1)
	pkey2 := make([]byte, 0, 32)
	var cid uint64

	encoded := <-recvCh
	decoded, err := obfs.Decode(encoded)
	if err != nil {
		return pkey2, cid, err
	}

	pkt, err := ParsePacket(decoded)
	if err != nil {
		return pkey2, cid, err
	}

	cid = pkt.ConnId
	pkey2 = append(pkey2, pkt.Payload[:32]...)

	// Send a OK packet to server with pkey2
	obfs.SetPkey(pkey2)

	pkt2 := NewAuthPacket(cid, pkey2)
	encoded = obfs.Encode(pkt2.AsBytes())
	if err := netio.WriteUDP(conn, encoded); err != nil {
		return pkey2, cid, err
	}

	return pkey2, cid, nil
}
*/


