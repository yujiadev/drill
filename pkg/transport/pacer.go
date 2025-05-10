package transport

import (
	//"container/heap"
	//"fmt"
)

type SendPacer struct {
	buf []byte
}

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

func (rp *RecvPacer) swap(index1, index2 int) {
	temp := rp.seq[index1]	
	rp.seq[index1] = rp.seq[index2]
	rp.seq[index2] = temp
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
		rp.heapifyUp(len(rp.seq)-1)
		return
	}
}

func (rp *RecvPacer) heapifyUp(index int) {
	// Reach the root node, return
	if index == 0 {
		return 
	}

	parent := (index-1)/2
	if rp.seq[parent] < rp.seq[index] {
		return
	}

	rp.swap(parent, index)
	rp.heapifyUp(parent)
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
		rp.heapifyDown(0)
	}

	return frame, true
}

func (rp *RecvPacer) heapifyDown(parent int) {
	if parent >= len(rp.seq) {
		return
	}

	left  := 2*parent+1	
	right := 2*parent+2
	bound := len(rp.seq)

	// Current parent node has two children
	if left < bound && right < bound {
		// Left node the smallest
		if rp.seq[left]	< rp.seq[right] && rp.seq[left] < rp.seq[parent] {
			rp.swap(left, parent)
			rp.heapifyDown(left)
			return
		}

		// Right node the smallest
		if rp.seq[right] < rp.seq[left] && rp.seq[right] < rp.seq[parent] {
			rp.swap(right, parent)
			rp.heapifyDown(right)
			return
		}

		// Parent node is the smallest (unlikely, but possible)
		if rp.seq[parent] < rp.seq[left] && rp.seq[parent] < rp.seq[right] {
			return
		}
	}

	// Current parent node only has left node 
	if left < bound && right >= bound {
		// Left node the smallest
		if rp.seq[left] < rp.seq[parent] {
			rp.swap(left, parent)
			rp.heapifyDown(left)
			return
		}

		// Parent node is the smallest
		return 
	}

	// Current parent node only has right node
	if right < bound && left >= bound {
		// Right node the smallest
		if rp.seq[right] < rp.seq[parent] {
			rp.swap(right, parent)
			rp.heapifyDown(right)
			return
		}

		return
	}
}