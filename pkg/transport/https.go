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

func handleConnection(
	conn net.Conn, 
	sendCh chan<-Frame, 
	chs *ChannelMap[Frame],
) {
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Printf("Err read HTTP request: %s\n", err)
		return
	}

	// Only allow HTTP CONNECT method
	if request.Method != "CONNECT" {
		log.Println("Err HTTP method: only HTTP CONNECT allowed")
		return
	}

	recvCh, src := chs.Create()
	connFrame := NewFrame(FCONN, 0, src, 0, []byte(string(request.Host)))

	sendCh <- connFrame
	respFrame := <-recvCh 

	if respFrame.Method != FOK {
		log.Printf("Err HTTP CONNECT request: can't fulfill\n")
		return
	}
	dst := respFrame.Destination

	// Notify client tunnel is established
    _, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
    if err != nil {
		log.Printf("Err notify HTTPS client: %s\n", err)
		return
    }

	notifyCh := make(chan Frame, 65535)

    var wg sync.WaitGroup
    wg.Add(2)

	go sendClient(conn, sendCh, notifyCh, src, dst, &wg)	
	go recvClient(conn, sendCh, recvCh, notifyCh, &wg)

	wg.Wait()
}


// client send to server
func sendClient(
	conn net.Conn, 
	sendCh chan<-Frame,
	notifyCh <-chan Frame, 
	src, dst uint64,
	wg *sync.WaitGroup,
) {
	seq := uint64(0)

	for {
		buf := make([]byte, 4096)
		n, err := conn.Read(buf)

		if err != nil {
			log.Printf("Err read (sendClient): %s\n", err)
			finFrame := NewFrame(FSENDDONE, seq+1, src, dst, []byte("senddone"))
			sendCh <- finFrame
			break
		}

		frame := NewFrame(FFWD, seq, src, dst, buf[:n])
		sendCh <- frame

		for {
			respFrame := <-notifyCh

			if respFrame.Method == FRECVDONE {
				wg.Done()
				return
			}

			seq += 1
			break
		}
	}

	wg.Done()
}

// client recv from server
func recvClient(
	conn net.Conn, 
	sendCh chan<-Frame,
	recvCh <-chan Frame,
	notifyCh chan<-Frame, 
	wg *sync.WaitGroup,
) {

	for {
		frame := <-recvCh

		if frame.Method == FACK || frame.Method == FRECVDONE {
			notifyCh <- frame
			continue
		}

		if frame.Method == FSENDDONE {
			break
		}

		log.Printf("client payload size (%v): %v\n", frame.Destination ,len(frame.Payload))

		err := WriteAllTCP(conn, frame.Payload)
		if err != nil {
			recvDone := NewFrame(
				FRECVDONE, 
				0, 
				frame.Destination, 
				frame.Source, 
				[]byte("recvdone"),
			)
			sendCh <- recvDone
			break
		}

		// Send ACK to sender
		ackFrame := NewFrame(
			FACK, 
			frame.Sequence, 
			frame.Destination, 
			frame.Source, 
			[]byte("ack"),
		)
		sendCh <- ackFrame
	}	

	wg.Done()
}
