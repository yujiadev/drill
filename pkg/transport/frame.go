package transport

import (
	"fmt"
	"time"
	"encoding/binary"
)

const (	
	FCONN byte = iota   // Connect
	FOK	                // OK
	FERR                // Err
	FFWD		        // Forward
	FRFWD		        // Retry forward
	FACK                // Ack
	FRACK               // Retry Ack
	FSENDDONE			// Send done
	FRECVDONE			// Recv done
)

// Method (1) + Time (8) + Seq (8) + Src (8) + Dst (8) + Payload size (4)
const FHEADERLEN int = 1 + 8 + 8 + 8 + 8 + 4

type Frame struct {
	Method byte
	Time int64
	Seq uint64
	Src uint64
	Dst uint64
	Payload []byte	
	Raw []byte
}

func NewFrame(method byte, seq, src, dst uint64, payload []byte) Frame {
	time := time.Now().Unix()
	payloadSize := uint32(len(payload))

	raw := []byte{ method }	
	raw, _ = binary.Append(raw, binary.BigEndian, uint64(time))
	raw, _ = binary.Append(raw, binary.BigEndian, seq)
	raw, _ = binary.Append(raw, binary.BigEndian, src)
	raw, _ = binary.Append(raw, binary.BigEndian, dst)
	raw, _ = binary.Append(raw, binary.BigEndian, payloadSize)
	raw    = append(raw, payload...)

	return Frame {
		method,
		time,
		seq,
		src,
		dst,
		payload,
		raw,
	}
}

func ParseFrame(data []byte) (Frame, error) {
	// Not enough byte to parse, return
	if len(data) < FHEADERLEN {
		return Frame{}, nil
	}

	method      := data[0]
	time        := int64(binary.BigEndian.Uint64(data[1:9]))
	seq         := binary.BigEndian.Uint64(data[9:17])
	src         := binary.BigEndian.Uint64(data[17:25])
	dst         := binary.BigEndian.Uint64(data[25:33])
	payloadSize := int(binary.BigEndian.Uint32(data[33:37]))

	if len(data[37:]) < payloadSize {
		return Frame{}, fmt.Errorf(
			"malformat payload size, want '%v', got '%v'",
			payloadSize,
			len(data[37:]),
		)
	}

	payload := data[37:37+payloadSize]
	raw     := data[0:37+payloadSize]

	return Frame{
		method,
		time,
		seq,
		src,
		dst,
		payload,
		raw,
	}, nil
}

func NewConnFrame(src uint64, addr string) Frame {
	return NewFrame(FCONN, 0, src, 0, []byte(addr))
}

func NewFwdFrame(seq, src, dst uint64, payload []byte) Frame {
	return NewFrame(FFWD, seq, src, dst, payload)
}

func NewRfwdFrame(seq, src, dst uint64, payload []byte) Frame {
	return NewFrame(FRFWD, seq, src, dst, payload)
}

func NewAckFrame(seq, src, dst uint64, payload []byte) Frame {
	return NewFrame(FACK, seq, src, dst, payload)
}

func NewSendDoneFrame(seq, src, dst uint64) Frame {
	return NewFrame(FSENDDONE, seq, src, dst, []byte("FSENDDONE"))	
}

func NewRecvDoneFrame(seq, src, dst uint64) Frame {
	return NewFrame(FRECVDONE, seq, src, dst, []byte("FRECVDONE"))
}
