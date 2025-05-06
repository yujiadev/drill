package transport

import (
	"fmt"
	"time"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"log"
	aead "golang.org/x/crypto/chacha20poly1305"

	"drill/pkg/xcrypto"
)

const (
	// Packet methods
	INIT byte = iota
	RETRY
	INIT2
	INITACK
	INITDONE
	TX
	FIN
	FINACK
)

type Negotiate struct {
	Time int64
	Id uint64
	Challenge []byte
	Answer []byte
	Key []byte
	Raw []byte
}

func NewNegotiate(id uint64, ans, key []byte, cphr *xcrypto.XCipher) Negotiate {
	now := uint64(time.Now().Unix())
	chall := make([]byte, 64)
	rand.Read(chall)

	// Fill with placeholders
	if len(ans) != 64 {
		ans = make([]byte, 64)
	}

	if len(key) != 32 {
		key = make([]byte, 32)
	}

	// Write id, challenge, timestamp, answer, key
	buf, _ := binary.Append(nil, binary.BigEndian, now)	
	buf, _ = binary.Append(buf, binary.BigEndian, id)
	buf = append(buf, chall...)
	buf = append(buf, ans...)
	buf = append(buf, key...)

	// Encrypt the neogitate content
	cphrtxt := cphr.Encrypt(&buf)

	return Negotiate {
		int64(now),
		id,
		chall,
		ans,
		key,
		cphrtxt,
	}
}

func ParseNegotiate(cphrtxt *[]byte, cphr *xcrypto.XCipher) (Negotiate, error) {
	// Number of bytes that plaintext should have
	const NEEDED int = 8+8+64+64+32
	plntxt, err := cphr.Decrypt(cphrtxt)

	if err != nil {
		err = errors.New("Decrypt Negotiate error: decrypt Negotiate failed")
		return Negotiate{}, err
	}

	// Check if the parsing is possible
	if len(plntxt) < NEEDED {
		msg := fmt.Sprintf(
			"Parse Negotiate error: insufficient bytes. got: %v, needed %v",
			len(plntxt),
			NEEDED,
		)
		return Negotiate{}, errors.New(msg)
	}

	time  := int64(binary.BigEndian.Uint64(plntxt[0:8]))
	id    := binary.BigEndian.Uint64(plntxt[8:16])
	chall := plntxt[16:80]
	ans   := plntxt[80:144]
	key   := plntxt[144:208]

	return Negotiate{
		time,
		id,
		chall,
		ans,
		key,
		plntxt,
	}, nil
}

type Packet struct {
	ConnectionId uint64
	Id uint64
	Method byte
	Token []byte
	Authenticate Negotiate
	Payload Frame
	Raw []byte
}

func NewPacket(
	cid, id uint64,
	method byte,
	token []byte,
	auth Negotiate,
	payload Frame,
	cphr *xcrypto.XCipher,
) Packet {
	// Encrypted the payload
	payloadBytes := cphr.Encrypt(&payload.Raw)

	tokenSize := uint32(len(token))
	authSize := uint32(len(auth.Raw))
	payloadSize := uint32(len(payloadBytes))

	raw, _:= binary.Append(nil, binary.BigEndian, cid)
	raw, _ = binary.Append(raw, binary.BigEndian, id)
	raw, _ = binary.Append(raw, binary.BigEndian, method)
	raw, _ = binary.Append(raw, binary.BigEndian, tokenSize)
	raw, _ = binary.Append(raw, binary.BigEndian, authSize)
	raw, _ = binary.Append(raw, binary.BigEndian, payloadSize)

	raw = append(raw, token...)
	raw = append(raw, auth.Raw...)
	raw = append(raw, payloadBytes...)

	return Packet {
		cid,
		id,
		method,
		token,
		auth,
		payload,
		raw,
	}
}

func ParsePacket(data *[]byte, cphr *xcrypto.XCipher) (Packet, error) {
	const NEEDED int = (8+8+1+4+4+4)

	if len(*data) < NEEDED {
		msg := fmt.Sprintf(
			"Parse Packet error: insufficient bytes. got: %v, needed %v",
			len(*data),
			NEEDED,
		)
		return Packet{}, errors.New(msg)
	}

	cid          := binary.BigEndian.Uint64((*data)[0:8])	
	id           := binary.BigEndian.Uint64((*data)[8:16])	
	method       := (*data)[16]
	tokenSize    := int(binary.BigEndian.Uint32((*data)[17:21]))
	authSize     := int(binary.BigEndian.Uint32((*data)[21:25]))
	payloadSize  := int(binary.BigEndian.Uint32((*data)[25:29]))

	if len((*data)[29:]) < tokenSize+authSize+payloadSize {
		msg := fmt.Sprintf(
			"Parse Packet error: insufficient bytes. got: %v, needed %v",
			len((*data)[29:]),
			tokenSize+authSize+payloadSize,
		)
		return Packet{}, errors.New(msg)
	}

	pvt          := 29
	token        := (*data)[pvt:pvt+tokenSize]
	authBytes    := (*data)[pvt+tokenSize:pvt+tokenSize+authSize]
	payloadBytes := (*data)[pvt+tokenSize+authSize:pvt+tokenSize+authSize+payloadSize]

	auth, err := ParseNegotiate(&authBytes, cphr)
	if err != nil {
		msg := fmt.Sprintf("Parse Packet error: %s", err)
		return Packet{}, errors.New(msg)
	}

	payloadBytes, err = cphr.Decrypt(&payloadBytes)
	if err != nil {
		msg := fmt.Sprintf("Parse Packet error: %s", err)
		return Packet{}, errors.New(msg)
	}

	payload, err := ParseFrame(&payloadBytes)
	if err != nil {
		msg := fmt.Sprintf("Parse Packet error: %s", err)
		return Packet{}, errors.New(msg)
	}

	return Packet{
		cid,
		id,
		method,
		token,
		auth,
		payload,
		*data,
	}, nil
}

func PeekConnectionId(data *[]byte) (uint64, error) {
	if len(*data) < 8 {
		return 0, errors.New("Parse error: insufficient bytes")
	}

	cid := binary.BigEndian.Uint64((*data)[0:8])
	return cid, nil
}

func NewInit(cphr *xcrypto.XCipher) Packet {
	padding := make([]byte, 1200)
	rand.Read(padding)

	return NewPacket(
		0,
		0,
		INIT,
		padding,
		Negotiate{},
		Frame{},
		cphr,
	)
}

func NewRetry(addr_str string, cphr *xcrypto.XCipher) Packet {
	// Generate the token
	now := time.Now().Unix()
	addr := []byte(addr_str) 
	rand_bytes := make([]byte, 16)
	rand.Read(rand_bytes)

	buf, err := binary.Append(nil, binary.BigEndian, now)
	buf = append(buf, addr...)
	buf = append(buf, rand_bytes...)

	// Encrypt the token
	key := make([]byte, aead.KeySize)
	nonce := make(
		[]byte,
		aead.NonceSize, 
		aead.NonceSize + len(buf) + aead.Overhead,
	)
	rand.Read(key)
	rand.Read(nonce)
	tcphr, err := aead.New(key)	

	if err != nil {
		log.Fatalf("can't init chacha20poly1305 cipher. %s", err)
	}

	token := tcphr.Seal(nonce, nonce, buf, nil)

	// Build Packet
	return NewPacket(
		0,
		0,
		RETRY,
		token,
		Negotiate{},
		Frame{},
		cphr,
	)
}

func NewInit2(id uint64, token []byte, cphr *xcrypto.XCipher) Packet {
	auth := NewNegotiate(id, []byte{}, []byte{}, cphr)
	return NewPacket(0, id, INIT2, token, auth, Frame{}, cphr)
}

func NewInitAck(cid, id uint64, ans, key[]byte, cphr *xcrypto.XCipher) Packet {
	auth := NewNegotiate(id, ans, key, cphr)
	return NewPacket(cid, id, INITACK, []byte{}, auth, Frame{}, cphr)
}

func NewInitDone(cid, id uint64, ans []byte, cphr *xcrypto.XCipher) Packet {
	auth := NewNegotiate(id, ans, []byte{}, cphr)
	return NewPacket(cid, id, INITDONE, []byte{}, auth, Frame{}, cphr)
}

func NewTx(cid, id uint64, payload Frame, cphr *xcrypto.XCipher) Packet {
	return NewPacket(cid, id, TX, []byte{}, Negotiate{}, payload, cphr)
}

func NewFin(cid, id uint64, cphr *xcrypto.XCipher) Packet {
	return NewPacket(cid, id, FIN, []byte{}, Negotiate{}, Frame{}, cphr)
}

func NewFinAck(cid, id uint64, cphr *xcrypto.XCipher) Packet {
	return NewPacket(cid, id, FINACK, []byte{}, Negotiate{}, Frame{}, cphr)
}