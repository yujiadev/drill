package transport 

import (
	"container/heap"
//	"fmt"
)

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

type SendPacer struct {
	WaitAck uint64	
	acks    SeqHeap
	frames  map[uint64]Frame
	wnd     uint64
	pvt     uint64
	src     uint64
	dst     uint64
	buf		[]byte
}

func NewSendPacer(src, dst uint64, wnd uint64) SendPacer {
	buf := []byte{}
	acks := SeqHeap{}
	frames := make(map[uint64]Frame)

	return SendPacer {
		0,
		acks,
		frames,
		wnd,
		0,
		src,
		dst,
		buf,	
	}
}

func (sp *SendPacer) PushBuffer(data []byte) {
	sp.buf = append(sp.buf, data...)
}

func (sp *SendPacer) PopFrame() (Frame, bool) {
	// Return if exceed windows size or not enough bytes to send
	if sp.pvt > sp.WaitAck + sp.wnd || len(sp.buf) == 0 {
		return Frame{}, false
	}

	payload := sp.buf[:min(len(sp.buf), 1024)]
	frame := NewFwdFrame(sp.pvt, sp.src, sp.dst, payload)

	sp.frames[sp.pvt] = frame
	sp.pvt += 1
	sp.buf = sp.buf[min(len(sp.buf), 1024):]	

	return frame, true
}

func (sp *SendPacer) RecvAck(recvAck uint64) {
	if recvAck < sp.WaitAck {
		return
	}
	delete(sp.frames, recvAck)

	heap.Push(&sp.acks, recvAck)

	// Try to remove all the consecutive ACKs, start from waitAck 
	for {
		if sp.WaitAck == sp.acks[0] {
			sp.WaitAck += 1	
			heap.Pop(&sp.acks)
			continue
		}

		break
	}
}

func (sp *SendPacer) GetBacklogSize() int {
	return len(sp.acks)
}

func (sp *SendPacer) GetBufferSize() int {
	return len(sp.buf)
}

func (sp *SendPacer) GetSelectiveRepeat() []Frame {
	repeat := []Frame{}

	for i := sp.WaitAck; i < sp.WaitAck + sp.wnd; i++ {
		frame, ok := sp.frames[i]
		if !ok {
			continue
		}

		repeat = append(repeat, frame)
	}

	return repeat
}

type RecvPacer struct {
	WaitSeq uint64
	seqs	SeqHeap
	frames  map[uint64]Frame
}

func NewRecvPacer() RecvPacer {
	seqs := []uint64{}
	frames := make(map[uint64]Frame)

	return RecvPacer {
		0,
		seqs,
		frames,
	}
}

func (rp *RecvPacer) PushFrame(frame Frame) {
	if frame.Seq < rp.WaitSeq {
		return
	}

	if _, ok := rp.frames[frame.Seq]; !ok {
		rp.frames[frame.Seq] = frame
		rp.seqs = append(rp.seqs, frame.Seq)
		return
	}
}

func (rp *RecvPacer) PopFrame() (Frame, bool) {
	if len(rp.seqs) == 0 || rp.WaitSeq != rp.seqs[0] {
		return Frame{}, false
	}

	seq := rp.seqs[0]
	heap.Pop(&rp.seqs)
	frame := rp.frames[seq]

	rp.WaitSeq += 1
	delete(rp.frames, seq)

	return frame, true
}