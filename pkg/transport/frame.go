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

func NewFrame(method byte, seq, src, dst uint64, payload []byte) {
	raw := []byte{ method }


}

func ParseFrame(data *[]byte) (Frame, error) {

	return Frame{}, nil
}