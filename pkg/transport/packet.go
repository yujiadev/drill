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
	"time"
	"crypto/rand"	
	"encoding/binary"
	"errors"
	aead "golang.org/x/crypto/chacha20poly1305"

	"drill/pkg/xcrypto"
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

// Generate token for Retry
func GenToken(raddr string) []byte {
	// Generate the token
	now := time.Now().Unix()
	addr := []byte(raddr)
	size := int32(len(addr))

	buf, err := binary.Append(nil, binary.BigEndian, now)
	if err != nil {
		panic("GenToken can't write 'now' to buffer")
	}

	buf, err = binary.Append(buf, binary.BigEndian, size)
	if err != nil {
		panic("GenToken can't write 'size' to buffer")
	}

	buf = append(buf, addr...)

	// Encrypt the token
	key := make([]byte, aead.KeySize)
	nonce := make(
		[]byte,
		aead.NonceSize, 
		aead.NonceSize + len(buf) + aead.Overhead,
	)
	rand.Read(key)
	rand.Read(nonce)
	cphr, err := aead.New(key)	

	if err != nil {
		panic("GenToken can't init chacha20poly1305 cipher")
	}

	token := cphr.Seal(nonce, nonce, buf, nil)
	return token
}

// Generate challenge for verifying remote identity
func GenChallenge(id uint64, raddr string, cphr *xcrypto.XCipher) {}

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

func NewRetry(addr string) Retry {
	token := GenToken(addr)
	return Retry { token }
}

func (pkt *Retry) ToBeBytes() []byte {
	size := int32(len(pkt.Token))
	buf, err := binary.Append(nil, binary.BigEndian, size)

	if err != nil {
		panic("Retry::ToBeBytes can't write 'size' to buffer")	
	}

	return append(buf, pkt.Token...)	
}

func RetryFromBeBytes(data *[]byte) (Retry, error) {
	if len(*data) < 4 {
		err := errors.New("RetryFromBeBytes not enough bytes to parse size out")
		return Retry {}, err
	}

    size := int(binary.BigEndian.Uint32((*data)[:4]))
    if len((*data)[4:]) < size {
 		err := errors.New("RetryFromBeBytes not enough bytes to parse token out")
		return Retry {}, err
    }

    return Retry { (*data)[4:(4+size)] }, nil
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
	Key []byte
}

type InitDone struct {
	Cid uint64
	Time int64
}