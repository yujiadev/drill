package transport

import (
	"fmt"
	"time"
	"encoding/binary"
	"errors"
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
	const NEEDED int = (1+8+8+8+8+4)

	if len(*data) < NEEDED {
		msg := fmt.Sprintf(
			"Parse Frame error: insufficient bytes. got: %v, needed %v",
			len(*data),
			NEEDED,
		)
		return Frame{}, errors.New(msg)
	}

	method := (*data)[0]
	time   := int64(binary.BigEndian.Uint64((*data)[1:9]))
	seq    := binary.BigEndian.Uint64((*data)[9:17])
	src    := binary.BigEndian.Uint64((*data)[17:25])
	dst    := binary.BigEndian.Uint64((*data)[25:33])
	
	payloadSize := int(binary.BigEndian.Uint32((*data)[33:37]))
	if len((*data)[37:]) < payloadSize {
		msg := fmt.Sprintf(
			"Parse Frame error: insufficient bytes. got: %v, needed %v",
			len(*data),
			payloadSize,
		)
		return Frame{}, errors.New(msg)
	}
	payload := (*data)[37:37+payloadSize]

	return Frame{
		method,
		time,
		seq,
		src,
		dst,
		payload,
		(*data),
	}, nil
}