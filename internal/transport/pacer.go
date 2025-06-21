package transport

import (
	"container/heap"
)

//
// A very basic mini heap, which will be used on the pacers as part of data
// struct that track the send/recv sequences
//
type SeqHeap []uint64

func (h SeqHeap) Len() int { return len(h) }
func (h SeqHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h SeqHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *SeqHeap) Push(x any) {
	*h = append(*h, x.(uint64))
}

func (h *SeqHeap) Pop() any {
	old := *h	
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

//
// Sender's pacer
//
type SendPacer struct {
	WaitAck 	uint64
	acks 		SeqHeap
	packets 	map[uint64] Packet
	wnd 		uint64
	Pvt 		uint64
	Cid 		uint64
	Src 		uint64
	Dst 		uint64
	buf 		[]byte
}

func NewSendPacer(cid, src, dst uint64) SendPacer {
	return SendPacer {
		0,
		[]uint64{},
		make(map[uint64]Packet),
		32,
		0,
		cid,
		src,
		dst,
		make([]byte, 0, 4096),
	}
}

func (sp *SendPacer) Push(data []byte) {
	sp.buf = append(sp.buf, data...)
}

func (sp *SendPacer) Pop() (Packet, bool) {
	// Return if exceed windows size or not enough byte to send
	if sp.Pvt >= sp.WaitAck + sp.wnd || len(sp.buf) == 0 {
		return Packet{}, false
	}

	// payload is at most 1024 bytes
	payload := sp.buf[:min(len(sp.buf), 512)]
	packet := NewFwdPacket(sp.Cid, sp.Pvt, sp.Src, sp.Dst, payload)

	// Track the frame
	sp.packets[sp.Pvt] = packet

	// Increment the pivot for future pops
	sp.Pvt += 1

	// Clean up buffer
	sp.buf = sp.buf[min(len(sp.buf), 512):]

	return packet, true
}

func (sp *SendPacer) Update(recvAck uint64) {
	// Ignore all the out of window ACKs
	if recvAck < sp.WaitAck || recvAck > sp.Pvt {
		return
	}	

	// Otherwise delete the track frame since receiver already have it.
	delete(sp.packets, recvAck)	

	// Start track the ACKs along with the previously received ACKs
	heap.Push(&sp.acks, recvAck)

	// Try to remove all the consecutive ACKs (slide window forward)
	for {
		if len(sp.acks) == 0 { break }

		// Always starting from the waiting ACK
		if sp.WaitAck == sp.acks[0] {
			sp.WaitAck += 1
			heap.Pop(&sp.acks)
			continue
		}

		break
	}
}

func (sp *SendPacer) ScaleUp() {
	if sp.wnd >= 2048 {
		return
	}

	sp.wnd = min(2048, sp.wnd*2)
}

func (sp *SendPacer) ScaleDown() {
	sp.wnd = max(32, uint64(float64(sp.wnd) * 0.7))
}

func (sp *SendPacer) IsWait() bool {
	if sp.WaitAck < sp.Pvt {
		return true
	}

	return false
}

func (sp *SendPacer) IsEmpty() bool {
	if len(sp.buf) == 0 {
		return true
	}

	return false
}

func (sp *SendPacer) Repeat() []Packet {
	packets := []Packet{}

	for i := sp.WaitAck; i < (sp.WaitAck+sp.wnd); i++ {
		packet, ok := sp.packets[i]

		if !ok { 
			continue 
		}

		packets = append(packets, packet)
	}

	return packets
}

func (sp *SendPacer) Done() Packet {
	return NewSendFinPacket(sp.Cid, sp.Pvt, sp.Src, sp.Dst)
}

//
// Receiver's pacer
//
type RecvPacer struct {
	WaitSeq 	uint64
	seqs 		SeqHeap
	packets		map[uint64]Packet
}

func NewRecvPacer() RecvPacer {
	return RecvPacer {
		0,
		[]uint64{},
		make(map[uint64]Packet),
	}
}

func (rp *RecvPacer) Push(pkt Packet) {
	if pkt.Seq < rp.WaitSeq {
		return
	}

	if _, exists := rp.packets[pkt.Seq]; !exists {
		rp.packets[pkt.Seq] = pkt
		heap.Push(&rp.seqs, pkt.Seq)
		return
	}
}

func (rp *RecvPacer) Pop() (Packet, bool) {
	// Return if no seq or let min seq still pending
	if len(rp.seqs) == 0 || rp.WaitSeq != rp.seqs[0] {
		return Packet{}, false
	}

	// Increment the wait sequence to the next
	rp.WaitSeq += 1

	// Pop the seq from the mini heap
	seq := (heap.Pop(&rp.seqs)).(uint64)

	// Get the frame
	packet := rp.packets[seq]

	// Clean up
	delete(rp.packets, seq)

	return packet, true
}

func (rp *RecvPacer) Fetch() []byte {
	buf := []byte{}

	for {
		packet, ok := rp.Pop()
		if !ok { 
			break 
		}

		buf = append(buf, packet.Payload...)
	}

	return buf
}