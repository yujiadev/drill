package transport

type SHOLD struct {

}

/*
package transport

import (
	"log"
	"fmt"
	"net"
	"time"
	"sync"
	"crypto/rand"
	"drill/internal/obfuscate"
	"drill/pkg/netio"
	//"drill/pkg/xcrypto"
)

type ServerTransport struct {
	laddr *net.UDPAddr
	pkey  []byte
	wg    *sync.WaitGroup
}

func NewServerTransport(
	laddr *net.UDPAddr,
	pkey []byte,
	wg *sync.WaitGroup,
) ServerTransport {
	return ServerTransport {
		laddr,
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
			go serverHandle(conn, raddr, sessions, pkey, data)
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
	pkey0 []byte,
	initData []byte,
) {
	recvCh, cid := sessions.Create(raddr)
	obfs := obfuscate.NewBasicObfuscator(pkey0)

	if err := serverRetry(conn, recvCh, raddr, pkey0); err != nil {
		log.Println(err)
		return 
	}

	pkey2 := make([]byte, 32)
	rand.Read(pkey2)
	if err := serverAuth(
		conn, 
		recvCh, 
		obfs, 
		raddr, 
		initData, 
		cid, pkey2,
	); err != nil {
		log.Println(err)
		return
	}

	//
	// Recv subsequent packets
	//
	obfs.SetPkey(pkey2)

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
		// Forward
		//
		if pkt.Method == FWD {
			ch, exists := endpoints.Get(pkt.ConnId)

			if !exists {
				continue
			}

			select {
			case ch <- pkt:
				break
			case <-time.After(20*time.Millisecond):
				break
			}
			
			continue
		}

		// 
		// Connect
		//
		if pkt.Method == CONN {
			recvCh, localId := endpoints.Create()
			//obfs2 := obfuscate.NewBasicObfuscator(pkey2)
			go serverConnTask(
				conn, 
				recvCh, 
				raddr, 
				endpoints, 
				localId, 
				pkey2, 
				pkt,
			)
			continue
		}
	}
}

func serverRetry(
	conn *net.UDPConn, 
	recvCh <-chan []byte,
	raddr *net.UDPAddr, 
	pkey0 []byte,
) error {
	token := NewRetryToken(raddr.IP, pkey0)

	if err := netio.WriteUDPAddr(conn, raddr, token); err != nil {
		return err
	}

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
	conn *net.UDPConn, 
	recvCh <-chan []byte,
	obfs obfuscate.Obfuscate, 
	raddr *net.UDPAddr, 
	initData []byte,
	cid uint64,
	pkey2 []byte,
) error {
	pkey1 := make([]byte, 0, 32)

	//
	// Get the pkey1 from the INIT packet from client.
	//
	decoded, err := obfs.Decode(initData)
	if err != nil {
		return err 
	}

	pkt, err := ParsePacket(decoded)
	if err != nil {
		return err
	}

	pkey1 = append(pkey1, pkt.Payload[0:32]...)

	//
	// Build a obfuscated packet based on the pkey1.
	// Send pkey2 to client
	//
	obfs.SetPkey(pkey1)

	pkt2 := NewAuthPacket(cid, pkey2)
	encoded := obfs.Encode(pkt2.AsBytes())
	if err := netio.WriteUDPAddr(conn, raddr, encoded); err != nil {
		return err
	}

	//
	// Recv a obfuscated packet that assoicated with pkey2
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

func serverConnTask(
	conn *net.UDPConn, 
	recvCh chan Packet, 
	raddr *net.UDPAddr, 
	endpoints *Endpoints,
	localId uint64, 
	pkey2 []byte, 
	connPkt Packet,
) {
	log.Println(connPkt)
}

func serverSendTask() {

}

func serverRecvTask() {

}
*/

