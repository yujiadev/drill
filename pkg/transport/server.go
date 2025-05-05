package transport

import (
	"fmt"
	"log"	
	//"time"
	"net"
	//"sync"

	"drill/pkg/xcrypto"
)

type Server struct {
	Address string
	Pkey string		
	Protocal string
}

func NewServer(
	host string,
	port uint16,
	pkey, protocal string,
) Server {
	addr := fmt.Sprintf("%s:%v", host, port)	

	return Server {
		addr,
		pkey,
		protocal,
	}
}

func (txp *Server) Run() {
	udpAddr, err := net.ResolveUDPAddr("udp", txp.Address)
	if err != nil {
		log.Fatalf("Error resolving UDP address: %v", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Error listening on UDP: %s", err)
	}
	fmt.Printf("Server is listening on %s\n", txp.Address)

	buf := make([]byte, 65535)
	cphr := xcrypto.NewXCipher(txp.Pkey)
	sessions := NewSessions()	
	sessions.IsConnectionExist(10)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP: %s", err)
			continue
		}

		go Authenticate(addr, buf[:n], cphr)
	}
}

func Authenticate(addr *net.UDPAddr, pkt []byte, cphr xcrypto.XCipher) {

	plntxt, err := cphr.Decrypt(&pkt)	
	if err != nil {
		log.Printf("Error Authenticating packet: %v", err)
		return
	}

	fmt.Printf(string(plntxt))
}
