package transport

import (
	"log"	
	"net"

	"drill/pkg/xcrypto"
)

type RecvPacket struct {
	Packet Packet
	Addr *net.UDPAddr
}

func GetSendRecvChannels() (chan Packet, chan RecvPacket){
	sendCh := make(chan Packet, 65535)
	recvCh := make(chan RecvPacket, 65535)

	return sendCh, recvCh
}

func SendUDP(
	cid, id uint64,
	conn *net.UDPConn, 
	ch <-chan Packet, 
	cphr xcrypto.XCipher,
) {

	for {
		packet := <-ch

		// Write all the bytes
		if err := WriteAllUDP(conn, packet.Raw); err != nil {
			log.Printf("Err send UDP: %s\n", err)
			continue
		}
	}
}

func WriteAllUDP(conn *net.UDPConn, data []byte) error {
	written := 0
	stop := len(data)

	for {
		n, err := conn.Write(data[written:])

		if err != nil {
			return err
		}

		written += n

		if written == stop {
			break
		}
	}

	return nil
}


func RecvUDP(
	cid, id uint64, 
	conn *net.UDPConn, 
	ch chan<- RecvPacket,
	cphr xcrypto.XCipher,
) {

	buf := make([]byte, 65535)
	for {
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Err recv UDP: %s\n", err)
			continue
		}

		bytes := buf[:n]	
		packet, err := ParsePacket(&bytes, &cphr)	
		if err != nil {
			log.Printf("Err parse Packet: %s\n", err)
			continue
		}

		ch <- RecvPacket {
			packet,
			raddr,
		}
	}
}
