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
	PINIT byte = iota
	PRETRY
	PINIT2
	PINITACK
	PINITDONE
	PTX
	PFIN
	PFINACK
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
	raw := cphr.Encrypt(buf)

	return Negotiate {
		int64(now),
		id,
		chall,
		ans,
		key,
		raw,
	}
}

func ParseNegotiate(cphrtxt []byte, cphr *xcrypto.XCipher) (Negotiate, error) {
	if len(cphrtxt) == 0 {
		return Negotiate{}, nil
	}

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
	key   := plntxt[144:176]

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
	data := cphr.Encrypt(payload.Raw)

	tokenSize := uint32(len(token))
	authSize := uint32(len(auth.Raw))
	payloadSize := uint32(len(data))

	raw, _:= binary.Append(nil, binary.BigEndian, cid)
	raw, _ = binary.Append(raw, binary.BigEndian, id)
	raw, _ = binary.Append(raw, binary.BigEndian, method)
	raw, _ = binary.Append(raw, binary.BigEndian, tokenSize)
	raw, _ = binary.Append(raw, binary.BigEndian, authSize)
	raw, _ = binary.Append(raw, binary.BigEndian, payloadSize)

	raw = append(raw, token...)
	raw = append(raw, auth.Raw...)
	raw = append(raw, data...)

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

func ParsePacket(data []byte, cphr *xcrypto.XCipher) (Packet, error) {
	// cid + id + method + tokenSize + authSize + payloadSize
	const NEEDED int = (8+8+1+4+4+4)

	if len(data) < NEEDED {
		err := fmt.Errorf(
			"insufficient bytes to parse value of sizes out. got: %v, needed %v",
			len(data),
			NEEDED,
		)
		return Packet{}, err
	}

	cid          := binary.BigEndian.Uint64(data[0:8])	
	id           := binary.BigEndian.Uint64(data[8:16])	
	method       := data[16]
	tokenSize    := int(binary.BigEndian.Uint32(data[17:21]))
	authSize     := int(binary.BigEndian.Uint32(data[21:25]))
	payloadSize  := int(binary.BigEndian.Uint32(data[25:29]))

	if len(data[29:]) < tokenSize+authSize+payloadSize {
		err := fmt.Errorf(
			"insufficient bytes to parse 'token/auth/payloadSize' (%v). got: %v, needed %v",
			method,
			len(data[29:]),
			tokenSize+authSize+payloadSize,
		)
		return Packet{}, err
	}

	pvt          := 29
	token        := data[pvt:pvt+tokenSize]
	authBytes    := data[pvt+tokenSize:pvt+tokenSize+authSize]
	payloadBytes := data[pvt+tokenSize+authSize:pvt+tokenSize+authSize+payloadSize]

	auth, err := ParseNegotiate(authBytes, cphr)
	if err != nil {
		return Packet{}, err
	}

	payloadBytes, err = cphr.Decrypt(payloadBytes)
	if err != nil {
		return Packet{}, err
	}

	payload, err := ParseFrame(&payloadBytes)
	if err != nil {
		return Packet{}, err
	}

	return Packet{
		cid,
		id,
		method,
		token,
		auth,
		payload,
		data,
	}, nil
}

func PeekConnectionId(data []byte) (uint64, error) {
	if len(data) < 8 {
		return 0, errors.New("Parse error: insufficient bytes")
	}

	cid := binary.BigEndian.Uint64(data[0:8])
	return cid, nil
}

func NewInit(cphr *xcrypto.XCipher) Packet {
	padding := make([]byte, 1200)
	rand.Read(padding)

	return NewPacket(
		0,
		0,
		PINIT,
		padding,
		Negotiate{},
		Frame{},
		cphr,
	)
}

func NewRetry(cid uint64, addr []byte, cphr *xcrypto.XCipher) Packet {
	// Generate the token
	now := time.Now().Unix()
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
		cid,
		0,
		PRETRY,
		token,
		Negotiate{},
		Frame{},
		cphr,
	)
}

func NewInit2(cid, id uint64, token []byte, cphr *xcrypto.XCipher) Packet {
	auth := NewNegotiate(id, []byte{}, []byte{}, cphr)
	return NewPacket(cid, id, PINIT2, token, auth, Frame{}, cphr)
}

func NewInitAck(cid, id uint64, ans, key[]byte, cphr *xcrypto.XCipher) Packet {
	auth := NewNegotiate(id, ans, key, cphr)
	return NewPacket(cid, id, PINITACK, []byte{}, auth, Frame{}, cphr)
}

func NewInitDone(cid, id uint64, ans []byte, cphr *xcrypto.XCipher) Packet {
	auth := NewNegotiate(id, ans, []byte{}, cphr)
	return NewPacket(cid, id, PINITDONE, []byte{}, auth, Frame{}, cphr)
}

func NewTx(cid, id uint64, payload Frame, cphr *xcrypto.XCipher) Packet {
	return NewPacket(cid, id, PTX, []byte{}, Negotiate{}, payload, cphr)
}

func NewFin(cid, id uint64, cphr *xcrypto.XCipher) Packet {
	return NewPacket(cid, id, PFIN, []byte{}, Negotiate{}, Frame{}, cphr)
}

func NewFinAck(cid, id uint64, cphr *xcrypto.XCipher) Packet {
	return NewPacket(cid, id, PFINACK, []byte{}, Negotiate{}, Frame{}, cphr)
}