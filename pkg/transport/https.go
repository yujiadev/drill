package transport

import (	
	"log"
	"bufio"
	"net"
	"net/http"
	"sync"
)

type HttpsProxy struct {
	addr *net.TCPAddr
	ch chan<-Frame
	chs *ChannelMap[Frame]
}

func NewHttpsProxy(
	addr *net.TCPAddr, 
	ch chan<-Frame, 
	chs *ChannelMap[Frame],
) HttpsProxy {

	return HttpsProxy {
		addr,
		ch,
		chs,
	}
}

func (pxy *HttpsProxy) Run() {	
	ln, err := 	net.ListenTCP("tcp", pxy.addr)
	if err != nil {
		log.Fatalf("Err listen on %s: %s\n", pxy.addr, err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Err accept HTTP request: %s\n")
			continue
		}

		go handleConnection(conn, pxy.ch, pxy.chs)
	}		
}

func handleConnection(conn net.Conn, ch chan<-Frame, chs *ChannelMap[Frame]) {
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Fatalf("Err read HTTP request: %s\n", err)
	}

	// Only allow HTTP CONNECT method
	if request.Method != "CONNECT" {
		log.Fatalf("Err HTTP method: only HTTP CONNECT allowed\n")
	}

	recvCh, src := chs.Create()
	connFrame := NewFrame(CONN, 0, src, 0, []byte(string(request.Host)))

	ch <- connFrame
	respFrame := <-recvCh 

	if respFrame.Method != OK {
		log.Fatalf("Err HTTP CONNECT request: can't fulfill\n")
	}
	dst := respFrame.Destination

	// Notify client tunnel is established
    _, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
    if err != nil {
		log.Fatalf("Err notify HTTPS client: %s\n", err)
    }

	ackCh := make(chan Frame, 65535)

    var wg sync.WaitGroup
    wg.Add(2)

	go sendClient(conn, ackCh, ch, src, dst, &wg)	
	go recvClient(conn, ackCh, ch, recvCh, &wg)

	wg.Wait()
}

// client send to server
func sendClient(
	conn net.Conn, 
	ackCh <-chan Frame, 
	sendCh chan<-Frame,
	src, dst uint64,
	wg *sync.WaitGroup,
) {
	seq := uint64(0)

	for {
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)

		if err != nil {
			log.Println("ERR")
			finFrame := NewFrame(DISCONN, seq+1, src, dst, []byte("disconn"))
			sendCh <- finFrame
			break
		}

		data := buf[:n]
		
		frame := NewFrame(FWD, seq, src, dst, data)
		sendCh <- frame
		log.Println("Sent More")

		ackFrame := <-ackCh
		
		if ackFrame.Sequence == seq {
			//seq += 1
			//break
		}

		seq += 1
		log.Println("Move on")
	}

	wg.Done()
}

// client recv from server
func recvClient(
	conn net.Conn, 
	ackCh chan<-Frame, 
	sendCh chan<-Frame,
	recvCh <-chan Frame,
	wg *sync.WaitGroup,
) {

	for {
		frame := <-recvCh

		if frame.Method == ACK {
			log.Println("RECV ACK")
			ackCh <- frame
			continue
		}

		if frame.Method == FIN {
			break
		}

		written := 0
		for {
			n, err := conn.Write(frame.Payload[written:])
			if err != nil {
				wg.Done()
				return
			}

			written += n
			if written == len(frame.Payload) {
				break
			}
		}

		// Send ACK to sender
		ackFrame := NewFrame(
			ACK, 
			frame.Sequence, 
			frame.Destination, 
			frame.Source, 
			[]byte("ack"),
		)
		sendCh <- ackFrame
	}	

	wg.Done()
}