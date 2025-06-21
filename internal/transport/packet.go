package transport

import (
	"fmt"
	"time"
	"crypto/rand"
	"encoding/binary"
)

const (
	INIT 		byte = iota
	RETRY	
	AUTH
	PING
	PONG
	FIN
	CONN
	FWD
	ACK	
	SENDFIN
	RECVFIN
	OK
	ERR
)

const NEEDED int = 8 + 1 + 8 + 8 + 8 + 8 + 4

type Packet struct {
	ConnId 		uint64
	Method 		byte
	Created 	time.Time
	Seq 		uint64
	Src 		uint64
	Dst 		uint64
	Payload 	[]byte
}

func NewPacket(
	cid uint64,
	method byte,
	seq, src, dst uint64,
	payload	[]byte,
) Packet {
	created := time.Now().Truncate(time.Second)

	payload_copy := make([]byte, 0, len(payload))
	payload_copy = append(payload_copy, payload...)

	return Packet {
		cid,
		method,
		created,
		seq,
		src,
		dst,
		payload_copy,
	}
}

func (pkt *Packet) AsBytes() []byte {
	payloadSize := len(pkt.Payload)	

	data := make([]byte, 0, NEEDED)

	// ConnID
	data, _ = binary.Append(data, binary.BigEndian, pkt.ConnId)

	// Method
	data = append(data, pkt.Method)

	// Created
	data, _ = binary.Append(data, binary.BigEndian, uint64(pkt.Created.Unix()))

	// Seq
	data, _ = binary.Append(data, binary.BigEndian, pkt.Seq)

	// Src
	data, _ = binary.Append(data, binary.BigEndian, pkt.Src)

	// Dst
	data, _ = binary.Append(data, binary.BigEndian, pkt.Dst)

	// Payload
	data, _ = binary.Append(data, binary.BigEndian, uint32(payloadSize))
	data = append(data, pkt.Payload...)

	return data
}

func ParsePacket(data []byte) (Packet, error) {
	if len(data) < NEEDED {
		return Packet{}, fmt.Errorf(
			"not enough bytes to parse ConnId, Method, Created, Seq, Src, Dst,"+
			" payloadSize, and Payload out for a Packet. got %v, want %v",
			len(data),
			NEEDED,
		)
	}

	// ConnId
	connId := binary.BigEndian.Uint64(data[0:8])

	// Method
	method := data[8]

	// Created
	created := time.Unix(int64(binary.BigEndian.Uint64(data[9:17])), 0)

	// Seq
	seq := binary.BigEndian.Uint64(data[17:25])

	// Src
	src := binary.BigEndian.Uint64(data[25:33])

	// Dst
	dst := binary.BigEndian.Uint64(data[33:41])

	// Payload
	payloadSize := int(binary.BigEndian.Uint32(data[41:45]))

	if len(data[45:]) < payloadSize {
		return Packet{}, fmt.Errorf(
			"not enough bytes to parse Payload out for a Packet." + 
			"got %v, want %v",
			len(data[45:]),
			payloadSize,
		)
	}

	payload := make([]byte, 0, payloadSize)
	payload = append(payload, data[45:45+payloadSize]...)

	return Packet {
		connId,
		method,
		created,
		seq,
		src,
		dst,
		payload,
	}, nil
}

func (pkt * Packet) ValidatePacket(
	cid uint64, 
	method byte, 
	seq, src, dst uint64,
) error {
	if cid != pkt.ConnId {
		return fmt.Errorf(
			"can't validate Packet.ConnId, want %v, got %v",
			cid,
			pkt.ConnId,
		)
	}

	if method != pkt.Method {
		return fmt.Errorf(
			"can't validate Packet.Method, want %v, got %v",
			method,
			pkt.Method,
		)
	}

	if time.Since(pkt.Created) > 1*time.Second {
		return fmt.Errorf("can't validate Packet.Created, expired")
	}

	if seq != pkt.Seq {
		return fmt.Errorf(
			"can't validate Packet.Seq, want %v, got %v",
			seq,
			pkt.Seq,
		)
	}

	if src != pkt.Src {
		return fmt.Errorf(
			"can't validate Packet.Src, want %v, got %v",
			src,
			pkt.Src,
		)	
	}

	if dst != pkt.Dst {
		return fmt.Errorf(
			"can't validate Packet.Dst, want %v, got %v",
			dst,
			pkt.Dst,
		)
	}

	return nil
}

func NewInitPacket(token []byte) Packet {
	if len(token) != 32 {
		panic("INIT packet token size needs to be 32 bytes")
	}

	padding := make([]byte, 1168)
	rand.Read(padding)	

	payload := make([]byte, 0, 1200)
	payload = append(payload, token...)
	payload = append(payload, padding...)

	return NewPacket(
		0,
		INIT,
		0,
		0,
		0,
		payload,
	)
}

func NewRetryPacket(token []byte) Packet {
	return NewPacket (
		0, 
		RETRY,
		0,
		0,
		0,
		token,
	)
}

func NewAuthPacket(cid uint64, token []byte) Packet {
	return NewPacket (
		cid,
		AUTH,
		0,
		0,
		0,
		token,
	)
}

func NewConnPacket(cid uint64, host string) Packet {
	return NewPacket (
		cid,
		CONN,
		0,
		0,
		0,
		[]byte(host),
	)
}

func NewOkPacket(cid uint64) Packet {
	return NewPacket(
		cid,
		OK,
		0,
		0,
		0,
		[]byte("OK"),
	)
}

func NewErrPacket(cid uint64) Packet {
	return NewPacket(
		cid,
		ERR,
		0,
		0,
		0,
		[]byte("ERR"),
	)
}

func NewFwdPacket(cid, seq, src, dst uint64, payload []byte) Packet {
	return NewPacket(
		cid,
		FWD,
		seq,
		src,
		dst,
		payload,
	)
}

func NewAckPacket(cid, seq, src, dst uint64) Packet {
	return NewPacket(
		cid,
		ACK,
		seq,
		src,
		dst,
		[]byte("ACK"),
	)
}

func NewSendFinPacket(cid, seq, src, dst uint64) Packet {
	return NewPacket(
		cid,
		SENDFIN,
		seq,
		src,
		dst,
		[]byte("SENDFIN"),
	)
}

func NewRecvFinPacket(cid, seq, src, dst uint64) Packet {
	return NewPacket(
		cid,
		RECVFIN,
		seq,
		src,
		dst,
		[]byte("RECVFIN"),
	)
}