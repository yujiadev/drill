package transport 

import (
	"log"
	"net"
)

type ClientTransport struct {
	laddr 		*net.TCPAddr
	raddr 		*net.UDPAddr	
	pkey 		string
	protocol 	string
}

func NewClientTransport(
	laddr *net.TCPAddr,
	raddr *net.UDPAddr,
	pkey, protocol string,
) ClientTransport {
	return ClientTransport {
		laddr,
		raddr,
		pkey,
		protocol,
	}
}

func (ct *ClientTransport) Run() {
	sendCh := make(chan Frame, 65535)
	delegateCh := make(chan ConfirmedConn, 65535)
	chMap := NewChannelMap[Frame]()
	conn, err := net.DialUDP("udp", nil, ct.raddr)

	if err != nil {
		log.Fatalf("Err dial %s: %s\n", ct.raddr, err)
	}

	go sendToServer(conn, sendCh)
	go recvFromServer(conn, chMap)
	go delegate() 

	// Start the HTTPS proxy
	https := NewHttpsProxy(
		ct.laddr,
		sendCh,
		delegateCh,
		chMap,
	)

	https.Run()
}

func sendToServer(conn *net.UDPConn, sendCh <-chan Frame) {
	for {
		frame := <-sendCh

		if err := WriteAllUDP(conn, frame.Raw); err != nil {
			log.Printf("Err send to server: %s\n", err)
			continue
		}
	}
}

func recvFromServer(conn *net.UDPConn, chMap *ChannelMap[Frame]) {
	buf := make([]byte, 65535)

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Err recv from server: %s\n", err)
			continue
		}

		frame, err := ParseFrame(buf[:n])
		if err != nil {
			log.Printf("Err parse recv frame from server: %s\n", err)
			continue
		}

		sendCh, exists := chMap.Get(frame.Dst)
		if !exists {
			log.Printf(
				"Err route frame from server: dst %v not exists\n",
				frame.Dst,
			)
			continue
		}

		sendCh <- frame
	}
}

func delegate() {

}

func delegateSend() {

}

func delegateRecv() {

}