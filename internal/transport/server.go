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

type ServerTransport struct {
	laddr 		*net.UDPAddr
	protocol 	string
	pkey  		[]byte
	wg    		*sync.WaitGroup
}

func NewServerTransport(
	laddr *net.UDPAddr,
	protocol string,
	pkey []byte,
	wg *sync.WaitGroup,
) ServerTransport {
	return ServerTransport {
		laddr,
		protocol,
		pkey,
		wg,
	}
}

func (st *ServerTransport) Run() {
	conn, err := net.ListenUDP("udp", st.laddr)

	// Return if something goes wrong during binding of address
	if err != nil {
		log.Printf("Error listen on UDP %s: %s\n", st.laddr, err)
		st.wg.Done()
		return
	}

	// Receive the ingress UDP packet
	sessions := NewSessions()
	buf := make([]byte, 65535)

	for {
		n, raddr, err := conn.ReadFromUDP(buf)

		if err != nil {
			continue
		}

		ch, exists := sessions.Get(raddr)

		if !exists && n < 1200 {
			continue
		}

		// Needs to deep copy, otherwise memory corruption
		data := make([]byte, 0, n)
		data = append(data, buf[:n]...)

		if !exists && n >= 1200 {
			pkey := make([]byte, 0, 32)
			pkey = append(pkey, st.pkey...)
			go serverHandle(conn, raddr, sessions, st.protocol, pkey, data)
			continue
		}

		// Set a time limit for the channel-sending operation. 
		select {
		case ch<-data:
			break
		case <-time.After(200*time.Millisecond):
			break
		}
	}
}

func serverHandle(
	conn *net.UDPConn,
	raddr *net.UDPAddr,
	sessions *Sessions,
	protocol string,
	pkey0 []byte,
	initBytes []byte,
) {
	recvCh, cid := sessions.Create(raddr)
	sendCh := make(chan []byte, 65535)

	go serverSocketSend(conn, raddr, sendCh)

	if err := serverRetry(sendCh, recvCh, raddr, pkey0); err != nil {
		log.Println(err)
		sessions.Delete(raddr)
		return
	}

	pkey2 := xcrypto.RandomKey(32) 

	if err := serverAuth(
		sendCh, 
		recvCh, 
		protocol, 
		pkey0, 
		pkey2, 
		cid, 
		initBytes,
	); err != nil {
		log.Println(err)
		sessions.Delete(raddr)
		return
	}

	//
	// Multiplexing and Forwarding
	//
	obfsCh := make(chan Packet, 65535)
	go serverObfsSend(
		obfsCh,
		sendCh,
		protocol,
		pkey2,
	)

	obfs := obfuscate.BuildObfuscator(protocol, pkey2)
	endpoints := NewEndpoints()

	for {
		data := <-recvCh

		decoded, err := obfs.Decode(data)
		if err != nil {
			log.Println(err)
			continue
		}

		pkt, err := ParsePacket(decoded)
		if err != nil {
			log.Println(err)
			continue
		}

		// 
		// Connect
		//
		if pkt.Method == CONN {
			ch, localId := endpoints.Create()
			go serverConn(
				obfsCh,
				ch, 
				endpoints, 
				cid,
				localId, 
				pkt,
			)
			continue
		}

		ch, exists := endpoints.Get(pkt.Dst)

		if !exists {
			continue
		}

		select {
		case ch <- pkt:
			break
		case <-time.After(200*time.Millisecond):
			break
		}
	}
}

func serverSocketSend(conn *net.UDPConn, raddr *net.UDPAddr, ch <-chan []byte) {
	for {
		data := <-ch
		if err := netio.WriteUDPAddr(conn, raddr, data); err != nil {
			continue
		}
	}
}

func serverObfsSend(
	recvCh <-chan Packet,
	sendCh chan<-[]byte, 
	protocol string, 
	pkey2 []byte,
) {
	obfs := obfuscate.BuildObfuscator(protocol, pkey2)

	for {
		pkt := <-recvCh

		encoded := obfs.Encode(pkt.AsBytes())

		sendCh <-encoded
	}
}

func serverRetry(
	sendCh chan<-[]byte, 
	recvCh <-chan []byte, 
	raddr *net.UDPAddr,
	pkey0 []byte,
) error {
	token := NewRetryToken(raddr.IP, pkey0)

	sendCh <- token

	select {
	case data :=<-recvCh:
		if !ValidateRetryToken(data, raddr.IP, pkey0) {
			return fmt.Errorf("can't validate retry token from client")
		}
		break
	case <-time.After(2*time.Second):
		return fmt.Errorf("timeout on receving retry token from client")
	}

	return nil
}

func serverAuth(
	sendCh chan<-[]byte, 
	recvCh <-chan []byte, 
	protocol string, 
	pkey0, pkey2 []byte, 
	cid uint64, 
	initBytes []byte,
) error {
	//
	// Get the pkey1 from the INIT packet from client.
	//
	obfs := obfuscate.BuildObfuscator(protocol, pkey0)
	pkey1 := make([]byte, 0, 32)
	decoded, err := obfs.Decode(initBytes)
	if err != nil {
		return err 
	}

	pkt, err := ParsePacket(decoded)
	if err != nil {
		return err
	}

	pkey1 = append(pkey1, pkt.Payload[0:32]...)

	//
	// Build a obfuscated packet (encrypted w/ pkey1)
	// Send it to client
	//
	obfs.SetPkey(pkey1)

	pkt = NewAuthPacket(cid, pkey2)
	encoded := obfs.Encode(pkt.AsBytes()) 
	sendCh <- encoded

	//
	// Recv a obfuscated packet that encrypted with pkey2
	//
	obfs.SetPkey(pkey2)

	encoded = <-recvCh

	decoded, err = obfs.Decode(encoded)
	if err != nil {
		return err
	}

	pkt, err = ParsePacket(decoded)
	if err != nil {
		return err
	}

	return nil
}

func serverConn(
	sendCh chan<-Packet, 
	recvCh <-chan Packet, 
	endpoints *Endpoints, 
	cid, localId uint64, 
	connPkt Packet,
) {
	remoteId := connPkt.Src
	host := string(connPkt.Payload)

	conn, err := net.Dial("tcp", host)
	if err != nil {
		errPkt := NewErrPacket(cid)
		errPkt.Dst = remoteId
		sendCh <- errPkt
		endpoints.Delete(localId)

		log.Printf("Error connect to %s: %s\n", host, err)
		return
	}

	okPkt := NewOkPacket(cid)
	okPkt.Src = localId
	okPkt.Dst = remoteId
	sendCh <- okPkt

	syncCh := make(chan Packet, 65535)

	var wg sync.WaitGroup
	wg.Add(2)
	//go SendTask(&wg, conn, sendCh, syncCh, cid, localId, remoteId)
	go SendTask2(&wg, conn, sendCh, syncCh, cid, localId, remoteId)

	go RecvTask(&wg, conn, sendCh, recvCh, syncCh, cid, localId, remoteId)
	wg.Wait()

	endpoints.Delete(localId)

	return
}
