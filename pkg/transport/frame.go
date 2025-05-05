package transport

import (
	//"fmt"
	"time"
	//"crypto/rand"
	"encoding/binary"
	//"errors"

)

const (
	CONN byte = iota
	DISCONN
	FWD
	ACK
)

type Frame struct {
	Method byte
	Time int64
	Sequence uint64
	Source uint64
	Destination uint64
	Payload []byte
	Raw []byte
}

func NewFrame(method byte, seq, src, dst uint64, payload []byte) Frame {
	time := uint64(time.Now().Unix())

	raw := []byte{ method }
	raw, _ = binary.Append(raw, binary.BigEndian, time)
	raw, _ = binary.Append(raw, binary.BigEndian, seq)
	raw, _ = binary.Append(raw, binary.BigEndian, src)
	raw, _ = binary.Append(raw, binary.BigEndian, dst)

	payloadSize := uint32(len(payload))
	raw, _ = binary.Append(raw, binary.BigEndian, payloadSize)
	raw = append(raw, payload...)

	return Frame {
		method,
		int64(time),
		seq,
		src,
		dst,
		payload,
		raw,
	}
}

func ParseFrame(data *[]byte) (Frame, error) {

	return Frame{}, nil
}