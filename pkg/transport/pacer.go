package transport

import (
	//"container/heap"
	"fmt"
)

//
// SendPacer
//
type SendPacer struct {
	buf []byte
	waitAck uint64
	acks []uint64
	frames map[uint64]Frame
	window uint64
	pivot uint64
	src uint64
	dst uint64
}

func NewSendPacer(src, dst uint64) SendPacer {
	buf := []byte{}	
	waitAck := uint64(0)
	acks := []uint64{}
	frames := make(map[uint64]Frame)
	window := uint64(64)
	pivot := uint64(0)

	return SendPacer {
		buf,
		waitAck,
		acks,
		frames,
		window,
		pivot,
		src,
		dst,
	}
}

func (sp *SendPacer) GetWaitAck() uint64 {
	return sp.waitAck
}

// Push incoming data into pacer's buffer
func (sp *SendPacer) PushBuffer(data []byte) {
	sp.buf = append(sp.buf, data...)
}

// Pop a frame for transmission
func (sp *SendPacer) PopFrame() (Frame, bool){
	// Return if out of window size or nothing can be popped
	if sp.pivot >= sp.waitAck + sp.window || len(sp.buf) == 0 {

		fmt.Printf(
			"sp.pivot = %v, sp.waitAck = %v, sp.window = %v, len(sp.buf) = %v"+
			" sp.acks[0] = %v, len(sp.acks) = %v\n",
			sp.pivot,
			sp.waitAck,
			sp.window,
			len(sp.buf),
			sp.acks[0],
			len(sp.acks),
		)

		return Frame{}, false
	}

	first1024 := sp.buf[:min(len(sp.buf), 1024)]
	frame := NewFrame(FFWD, sp.pivot, sp.src, sp.dst, first1024)

	sp.frames[sp.pivot] = frame
	sp.pivot += 1
	sp.buf = sp.buf[min(len(sp.buf), 1024):]

	return frame, true
}

func (sp *SendPacer) RecvAck(ack uint64) {
	// Non-exist ack, return
	if _, ok := sp.frames[ack]; !ok {
		return
	}
	delete(sp.frames, ack)

	sp.acks = append(sp.acks, ack)
	heapifyUp(&sp.acks, len(sp.acks)-1)

	// Remove all the consective acks, start from waitAck
	for {
		// The lower end of windows can move forward 
		if sp.waitAck == sp.acks[0] {
			sp.waitAck += 1

			// Only one ack left, empty it then return
			if len(sp.acks) == 1 {
				sp.acks = []uint64{}
				break
			}

			// Two acks left, pop the min one then recheck
			if len(sp.acks) == 2 {
				sp.acks = sp.acks[1:]
				continue
			}

			// Three or more acks left, pop the min one, then heapify down
			last := sp.acks[len(sp.acks)-1]
			sp.acks[0] = last 
			sp.acks = sp.acks[:len(sp.acks)-1]

			heapifyDown(&sp.acks, 0)
			continue
		}

		break
	}	

	return
}

//
// RecvPacer
//
type RecvPacer struct {
	waitSeq uint64
	seq []uint64				// Acknowledged sequences in a binary heap
	frames map[uint64]Frame 	// Map that stored recv frames
}

func NewRecvPacer() RecvPacer {
	waitSeq := uint64(0)
	seq := []uint64{}
	frames := make(map[uint64]Frame)

	return RecvPacer {
		waitSeq,
		seq,
		frames,
	}
}

func (rp *RecvPacer) PushFrame(frame Frame) {
	// Sequence of recv frame is less than the one that is waiting for, return
	if frame.Sequence < rp.waitSeq {
		return
	}

	if _, ok := rp.frames[frame.Sequence]; !ok {
		// Store the frame for later usage
		rp.frames[frame.Sequence] = frame

		// Track the sequence of recv frame
		rp.seq = append(rp.seq, frame.Sequence)

		// Sort the heap
		heapifyUp(&rp.seq, len(rp.seq)-1)
		return
	}
}

func (rp *RecvPacer) PopFrame() (Frame, bool) {
	// Return nothing in case of empty sequence or wait sequence still pending
	if len(rp.seq) == 0 || rp.waitSeq != rp.seq[0] {
		return Frame{}, false
	}

	// Pop the sequence	
	lastSeq := rp.seq[len(rp.seq)-1]
	rp.seq[0] = lastSeq
	rp.seq = rp.seq[:len(rp.seq)-1]

	// Retrive the frame, then remove the frame from storage
	frame, _ := rp.frames[rp.waitSeq]
	delete(rp.frames, rp.waitSeq)

	// Update current wait sequence to the next one
	rp.waitSeq += 1

	// Compare if there are at least two sequences in track
	if len(rp.seq) >= 2 {
		heapifyDown(&rp.seq, 0)
	}

	return frame, true
}

func heapifyUp(heap *[]uint64, index int) {
	// Reach the root node, return
	if index == 0 {
		return 
	}

	parent := (index-1)/2
	if (*heap)[parent] < (*heap)[index] {
		return
	}

	swap(heap, parent, index)
	heapifyUp(heap, parent)	
}

func heapifyDown(heap *[]uint64, parent int) {
	if parent >= len(*heap) {
		return
	}

	left  := 2*parent+1
	right := 2*parent+2
	bound := len(*heap)

	// Current parent node has two children
	if left < bound && right < bound {
		// Left node the smallest
		if (*heap)[left] < (*heap)[right] && (*heap)[left] < (*heap)[parent] {
			swap(heap, left, parent)
			heapifyDown(heap, left)
			return
		}

		// Right node the smallest
		if (*heap)[right] < (*heap)[left] && (*heap)[right] < (*heap)[parent] {
			swap(heap, right, parent)
			heapifyDown(heap, right)
			return
		}

		// Parent node is the smallest (unlikely, but possible)
		if (*heap)[parent] < (*heap)[left] && (*heap)[parent] < (*heap)[right] {
			return
		}
	}

	// Current parent node only has left node 
	if left < bound && right >= bound {
		// Left node the smallest
		if (*heap)[left] < (*heap)[parent] {
			swap(heap, left, parent)
			heapifyDown(heap, left)
			return
		}

		// Parent node is the smallest
		return 
	}

	// Current parent node only has right node
	if right < bound && left >= bound {
		// Right node the smallest
		if (*heap)[right] < (*heap)[parent] {
			swap(heap, right, parent)
			heapifyDown(heap, right)
			return
		}

		return
	}
}

func swap(heap *[]uint64, index1, index2 int) {
	temp := (*heap)[index1]	
	(*heap)[index1] = (*heap)[index2]
	(*heap)[index2] = temp
}