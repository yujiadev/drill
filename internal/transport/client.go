package transport

import (
	"log"
	"fmt"
	"time"
	"net"
	"sync"
	"drill/internal/obfuscate"
	"drill/pkg/netio"
	"drill/pkg/xcrypto"
)

type ClientTransport struct {
	laddr 		*net.TCPAddr
	raddr 		*net.UDPAddr
	protocol 	string
	pkey  		[]byte
	wg    	*sync.WaitGroup
}

func NewClientTransport(
	laddr *net.TCPAddr,
	raddr *net.UDPAddr,
	protocol string,
	pkey  []byte,
	wg    *sync.WaitGroup,
) ClientTransport {
	return ClientTransport {
		laddr,
		raddr,
		protocol,
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

	go clientSocketSend(conn, sendCh)
	go clientSocketRecv(conn, recvCh)

	pkey2, cid, err := ct.clientHandshake(sendCh, recvCh)
	if err != nil {
		log.Printf("Error on handshake. %s\n", err)
		ct.wg.Done()
		return
	}

	obfsCh := make(chan Packet, 65535)
	endpoints := NewEndpoints()

	var wg sync.WaitGroup
	wg.Add(1)

	go clientObfsSend(
		obfsCh, 
		sendCh,
		ct.protocol, 
		pkey2, 
		cid,
	)

	go clientObfsRecv(
		endpoints,
		recvCh,
		ct.protocol, 
		pkey2, 
		cid,
	)

	go clientHttpsProxy(
		endpoints,
		obfsCh,
		ct.laddr, 
		cid,
	)

	wg.Wait()
}

func clientSocketSend(conn *net.UDPConn, ch <-chan []byte) {
	for {
		select {
		case data :=<-ch:
			if err := netio.WriteUDP(conn, data); err != nil {
				log.Printf("Error send data to socket. %s\n", err)
				continue
			}
			break
		case <-time.After(2000*time.Millisecond):
			break
		}
	}
}

func clientSocketRecv(conn *net.UDPConn, ch chan<-[]byte) {
	buf := make([]byte, 65535)

	for {
		n, _, err := conn.ReadFromUDP(buf)

		if err != nil {
			log.Printf("Error recv data from socket. %s\n", err)
			continue
		}
		
		data := make([]byte, 0, n)
		data = append(data, buf[:n]...)
		ch <- data
	}
}

func (ct *ClientTransport) clientHandshake(
	sendCh chan <-[]byte,
	recvCh <-chan []byte,
) ([]byte, uint64, error) {
	pkey1 := ct.clientInit(sendCh)

	if err := ct.clientRetry(sendCh, recvCh); err != nil {
		return []byte{}, 0, err
	}

	pkey2, cid, err := ct.clientAuth(sendCh, recvCh, pkey1)
	if err != nil {
		return []byte{}, 0, err
	}

	return pkey2, cid, nil
}

func (ct *ClientTransport) clientInit(sendCh chan <- []byte) []byte {
	pkey1 := xcrypto.RandomKey(32)
	obfs := obfuscate.BuildObfuscator(ct.protocol, ct.pkey)

	pkt := NewInitPacket(pkey1)
	encoded := obfs.Encode(pkt.AsBytes())

	sendCh <-encoded

	return pkey1
}

func (ct *ClientTransport) clientRetry(
	sendCh chan <-[]byte,
	recvCh <-chan []byte,
) error {
	select {
	case data :=<-recvCh:
		sendCh <- data
		return nil 
	case <-time.After(2*time.Second):
		return fmt.Errorf("timeout on receving retry token from server")
	}	
}

func (ct *ClientTransport) clientAuth(
	sendCh chan <-[]byte,
	recvCh <-chan []byte,
	pkey1 []byte,
) ([]byte, uint64, error) {
	//
	// Recv AUTH packet from server, try to get the pkey2 from server
	//
	obfs := obfuscate.BuildObfuscator(ct.protocol, pkey1)

	encoded := <-recvCh
	decoded, err := obfs.Decode(encoded)
	if err != nil {
		log.Println(err)
		return []byte{}, 0, err
	}

	pkt, err := ParsePacket(decoded)
	if err != nil {
		return []byte{}, 0, err
	}

	cid := pkt.ConnId
	pkey2 := append([]byte{}, pkt.Payload[:32]...)

	//
	// Send a OK packet (encrypted with pkey2) to server as acknowledgement
	//
	obfs.SetPkey(pkey2)
	pkt = NewOkPacket(cid)
	encoded = obfs.Encode(pkt.AsBytes())
	sendCh <- encoded

	return pkey2, cid, nil
}

func clientObfsSend(
	recvCh <-chan Packet,
	sendCh chan<-[]byte, 
	protocol string, 
	pkey2 []byte, 
	cid uint64,
) {
	obfs := obfuscate.BuildObfuscator(protocol, pkey2)

	for {
		pkt := <-recvCh

		encoded := obfs.Encode(pkt.AsBytes())

		sendCh <-encoded
	}	
}

func clientObfsRecv(
	endpoints *Endpoints,
	recvCh <-chan []byte,
	protocol string, 
	pkey2 []byte, 
	cid uint64,
) {
	obfs := obfuscate.BuildObfuscator(protocol, pkey2)

	for {
		encoded := <-recvCh
	
		decoded, err := obfs.Decode(encoded)
		if err != nil {
			log.Printf("Err on deobfuscating encoded data. %s\n", err)
			continue
		}

		pkt, err := ParsePacket(decoded)
		if err != nil {
			log.Printf("Err on parsing the decoded data to a packet. %s\n", err)
			continue
		}

		ch, exists := endpoints.Get(pkt.Dst)

		if !exists {
			log.Printf("Not found dst %v\n", pkt.Dst)
			continue
		}

		ch <-pkt
	}
}

func clientHttpsProxy(
	endpoints *Endpoints,
	obfsCh chan<-Packet,
	laddr *net.TCPAddr,
	cid uint64, 
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

		go clientHandle(endpoints, obfsCh, conn, cid)
	}
}

func clientHandle(
	endpoints *Endpoints,
	obfsCh chan<-Packet,
	conn net.Conn, 
	cid uint64,
) {
	host, err := ParseHTTPConnectHost(conn)
	if err != nil {
		log.Printf("Err parse HTTP CONNECT host: %s\n", host)
		return
	}

	recvCh, localId := endpoints.Create()
	connPkt := NewConnPacket(cid, host)
	connPkt.Src = localId

	obfsCh <- connPkt
	recvPkt := <-recvCh

	if recvPkt.Method != OK {
		if err := NotifyClientOnFailure(conn); err != nil {
			log.Printf("Err notify client on faiure: %s\n", err)	
		}
		endpoints.Delete(localId)

		return
	}

	if err := NotifyClientOnSuccess(conn); err != nil {
		log.Printf("Err notify client on success: %s\n", err)	
		endpoints.Delete(localId)
		return
	}

	remoteId := recvPkt.Src
	syncCh := make(chan Packet, 65535)

	var wg sync.WaitGroup
	wg.Add(2)
	//go SendTask(&wg, conn, obfsCh, syncCh, cid, localId, remoteId)
	go SendTask2(&wg, conn, obfsCh, syncCh, cid, localId, remoteId)
	go RecvTask(&wg, conn, obfsCh, recvCh, syncCh, cid, localId, remoteId)
	wg.Wait()

	endpoints.Delete(localId)

	return
}
