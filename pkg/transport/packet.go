/*
	Packet Flow:
	Client                    Server
	  | -------- INIT ---------> |
	  | <------- RETRY --------> |
	  | -------- INIT2 --------> |
	  | <------- INITACK ------  |
	  | -------- INITDONE -----> |
	  | <------- TX -----------> |
	  | <------- TX -----------> |
	  | <------- TX -----------> |
	  | -------- FIN ----------> |
	  | <------- FINACK -------- |
*/

package transport

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
)

const (
	INIT uint64 = iota
	RETRY	
	INIT2
	INITACK
	INITDONE
	TX
	FIN
	FINACK
)

func GenToken() {}

func GenChallenge() {}

func GenAnswer() {}

type Init struct {
	Padding []byte
}

func NewInit() Init {
	padding := make([]byte, 1200)
	rand.Read(padding)

	return Init { padding }
}

func (pkt *Init) ToBeBytes() []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, INIT)
	return append(buf, pkt.Padding...)
}

func InitFromBeBytes(data *[]byte) (Init, error) {
	if len(*data) < (8 + 1200) {
		return Init{}, errors.New("Init::FromBeBytes not enough bytes to parse")
	}

	method := binary.BigEndian.Uint64((*data)[:8])
	if method != INIT {
		return Init{}, errors.New("Init::FromBeBytes not INIT method")
	}

	padding := (*data)[8:1208]
	init := Init { padding }

	return init, nil
}


type Retry struct {
	Token []byte
}

type Init2 struct {
	Id uint64
	Token []byte
	Challenge []byte
}

type InitAck struct {
	Cid uint64
	Time int64
	Answer []byte
}

type InitDone struct {
	Cid uint64
	Time int64
}