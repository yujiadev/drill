package transport

import (
	"time"
	"net"
	"sync"
	"context"
	"drill/pkg/netio"
)

func SendTask(
	wg *sync.WaitGroup, 
	conn net.Conn, 
	sendCh chan<-Packet, 
	syncCh <-chan Packet,
	cid, localId, remoteId uint64, 
) {
	connCh := make(chan[]byte, 65535)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go netio.TCPReadAsChannel(ctx, conn, connCh)	

	pacer := NewSendPacer(cid, localId, remoteId)
	duration := 300*time.Millisecond

	// 
	// Stop & Wait for the time being
	//
	for {
		if pacer.IsEmpty() {

			data := <-connCh
		
			if len(data) == 0 {
				sendCh <- pacer.Done()
				wg.Done()
				return 
			}

			pacer.Push(data)
		} 

		packet, _ := pacer.Pop()
		sendCh <- packet
		
		for isWait := true; isWait; {
			select {
			case pkt := <-syncCh:
				if pkt.Method == ACK {
					pacer.Update(pkt.Seq)
					isWait = false
					break
				}

				if pkt.Method == RECVFIN {
					wg.Done()
					return
				}

				break
			case <-time.After(duration):
				for _, pkt := range pacer.Repeat() {
					sendCh <- pkt
				}
				break
			}
		}
	}
}

func RecvTask(
	wg *sync.WaitGroup, 
	conn net.Conn, 
	sendCh chan<-Packet, 
	recvCh <-chan Packet, 
	syncCh chan<-Packet,
	cid, localId, remoteId uint64, 
) {
	pacer := NewRecvPacer()

	for {
		packet := <-recvCh 

		if packet.Method == ACK || packet.Method == RECVFIN {
			syncCh <- packet
			continue
		}

		if packet.Method != FWD {
			wg.Done()
			return
		}

		pacer.Push(packet)
		ackPkt := NewAckPacket(cid, packet.Seq, localId, remoteId)
		sendCh <- ackPkt

		data := pacer.Fetch()
		if len(data) == 0 {
			continue
		}

		if err := netio.WriteTCP(conn, data); err != nil {
			recvFinPkt := NewRecvFinPacket(cid, 0, localId, remoteId)
			sendCh <- recvFinPkt
			wg.Done()
			return
		}
	}
}