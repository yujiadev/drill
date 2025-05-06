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
	conns := make(map[uint64] chan []byte)
	counter := uint64(1)

	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP: %s", err)
			continue
		}

		cid, err := PeekConnectionId(&buf)
		if err != nil {
			log.Printf("Error peeking recv UDP packet: %s", err)
			continue
		}

		tx, ok := conns[cid]
		if ok {
			tx <- buf[0:n]
			continue
		}

		ch := make(chan []byte, 65535)
		new_cid := counter
		counter += 1
		go HandleNewConnection(addr, buf[0:n], new_cid, ch)
	}
}

func HandleNewConnection(
	addr *net.UDPAddr, 
	data []byte, 
	cid uint64, 
	ch <-chan []byte,
) {
	cphr := xcrypto.NewXCipher("7abY7sBqNrtN5Z+NElo19hBDO1ixZ1+EGrrMq0gAjeE=")
	_, err := ParsePacket(&data, &cphr)

	if err != nil {
		log.Printf("Error parse new connection Packet: %s", err)
		return
	}	
}