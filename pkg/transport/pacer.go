package transport

import (
	//"container/heap"
	"fmt"
)

type SendPacer struct {

}

type RecvPacer struct {
	Sequence []uint64
}

func NewRecvPacer() RecvPacer {
	seq := []uint64{}
	return RecvPacer {
		seq,
	}
}

func (rp *RecvPacer) Push(num uint64) {
	rp.Sequence = append(rp.Sequence, num)
	lastIndex := len(rp.Sequence)-1

	rp.insertSort(lastIndex)
}

func (rp *RecvPacer) insertSort(index int) {
	// At root, return
	if index == 0 {
		return
	}

	parentIndex := (index-1)/2
	if rp.Sequence[parentIndex] < rp.Sequence[index] {
		return
	}

	rp.Swap(parentIndex, index)
	rp.insertSort(parentIndex)
}

func (rp * RecvPacer) Pop() (uint64, error) {
	if len(rp.Sequence)	== 0 {
		return 0, fmt.Errorf("empty seqence")
	}

	if len(rp.Sequence) == 1 {
		seq := rp.Sequence[0]
		rp.Sequence = []uint64{}
		return seq, nil
	}

	if len(rp.Sequence) == 2 {
		seq := rp.Sequence[0]
		rp.Sequence = rp.Sequence[1:]
		return seq, nil
	}

	seq := rp.Sequence[0]

	// Put the last seq on the root
	last := rp.Sequence[len(rp.Sequence)-1]
	rp.Sequence[0] = last

	 // Remove the extra seq
	rp.Sequence = rp.Sequence[:len(rp.Sequence)-1]

	rp.removeSort(0)

	return seq, nil
}

func (rp *RecvPacer) removeSort(parent int) {
	left  := 2*parent+1
	right := 2*parent+2
	bound := len(rp.Sequence) 

	//fmt.Printf("parent: %v, left: %v, right: %v, bound: %v\n", parent, left, right, bound)

	// At leaf
	if parent >= bound || (right >= bound && left >= bound) {
		return
	}

	// Compare to left leaf, left leaf is the smallest
	if right >= bound && left < bound && rp.Sequence[left] < rp.Sequence[parent] {
		rp.Swap(left, parent)
		return
	}

	// Compare to left leaf, parent is the smallest
	if right >= bound && left < bound && rp.Sequence[parent] < rp.Sequence[left] {
		return
	}

	// Compare to right leaf, right leaf is the smallest
	if left >= bound && right < bound && rp.Sequence[right] < rp.Sequence[parent] {
		rp.Swap(right, parent)
		return 
	}

	// Compare to right leaf, parent is the smallest
	if left >= bound && right < bound && rp.Sequence[parent] < rp.Sequence[right] {
		return 
	}

	// Left is the smallest child
	if rp.Sequence[left] < rp.Sequence[right] && rp.Sequence[left] < rp.Sequence[parent] {
		rp.Swap(left, parent)
		rp.removeSort(left)
		return
	}

	// Right is the smallest child
	if rp.Sequence[right] < rp.Sequence[left] && rp.Sequence[right] < rp.Sequence[parent] {
		rp.Swap(right, parent)
		rp.removeSort(right)
		return
	}

	// Parent is the smallest
	return
}

func (rp *RecvPacer) Swap(index1, index2 int) {
	temp := rp.Sequence[index1]	

	rp.Sequence[index1] = rp.Sequence[index2]
	rp.Sequence[index2] = temp
}