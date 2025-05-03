/*
	Packet Flow:
	Client                    Server
	  | -------- INIT ---------> |
	  | <------- RETRY --------- |
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
	size := uint32(len(addr))

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
func GenChallenge(id uint64, raddr string, cphr *xcrypto.XCipher) []byte {	
	now := time.Now().Unix()
	addr := []byte(raddr)
	size := uint32(len(addr))
	msg := make([]byte, 64)
	rand.Read(msg)

	buf, err := binary.Append(nil, binary.BigEndian, id)
	if err != nil {
		panic("GenChallenge can't write 'id' to buffer")
	}

	buf, err = binary.Append(buf, binary.BigEndian, now)
	if err != nil {
		panic("GenChallenge can't write 'now' to buffer")
	}

	buf, err = binary.Append(buf, binary.BigEndian, size)
	if err != nil {
		panic("GenChallenge can't write 'size' to buffer")
	}

	buf = append(buf, addr...)
	buf = append(buf, msg...)

	return cphr.Encrypt(&buf)
}

// Return (timestamp, host address, id, message, error)
func SolveChallege(
	data *[]byte, 
	cphr *xcrypto.XCipher,
) (int64, string, uint64, []byte, error) {
	var tmsp int64
	var host string
	var id uint64
	msg := make([]byte, 0)
	chall, err := cphr.Decrypt(data)

	if err != nil {
		return tmsp, host, id, msg, err
	}

	// Parse id
	if len(chall) < 8 {
		panic("SolveChallege not enough bytes to parse 'id'")
	}
	id = binary.BigEndian.Uint64(chall[0:8])

	// Parse timestamp
	if len(chall[8:]) < 8 {
		panic("SolveChallege not enough bytes to parse 'now'")
	}
	tmsp = int64(binary.BigEndian.Uint64(chall[8:16]))

	// Parse address size
	if len(chall[16:]) < 4 {
		panic("SolveChallege not enough bytes to parse 'size'")
	}
	addr_size := int(binary.BigEndian.Uint32(chall[16:20]))

	// Parse address
	if len(chall[20:]) < addr_size {
		panic("SolveChallege not enough bytes to parse 'addr'")
	}
	host = string(chall[20:20+addr_size])

	// Parse message
	if len(chall[20+addr_size:]) < 64 {
		panic("SolveChallege not enough bytes to parse 'msg'")
	}
	msg = chall[20+addr_size:20+addr_size+64]

	return tmsp, host, id, msg, nil
}

//
// Init
//
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

//
// Retry
//
type Retry struct {
	Token []byte
}

func NewRetry(addr string) Retry {
	token := GenToken(addr)
	return Retry { token }
}

func (pkt *Retry) ToBeBytes() []byte {
	size := uint32(len(pkt.Token))
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

//
// Init2
//
type Init2 struct {
	Id uint64
	Token []byte
	Challenge []byte
}

//
// InitAck
//
type InitAck struct {
	Cid uint64
	Time int64
	Answer []byte
	Key []byte
}

//
// InitDone
//
type InitDone struct {
	Cid uint64
	Time int64
}