package transport

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

import (
	"log"
	"errors"
	"time"
	"crypto/rand"	
	"encoding/binary"
	aead "golang.org/x/crypto/chacha20poly1305"

	"drill/pkg/xcrypto"
)

const (
	INIT byte = iota
	RETRY	
	INIT2
	INITACK
	INITDONE
	TX
	FIN
	FINACK
)

func NewToken(addr_str string) []byte {
	// Generate the token
	now := time.Now().Unix()
	addr := []byte(addr_str) 
	rand_bytes := make([]byte, 16)
	rand.Read(rand_bytes)

	buf, err := binary.Append(nil, binary.BigEndian, now)
	if err != nil {
		log.Fatalf("can't append 'now' to buf. %s", err)
	}

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
	cphr, err := aead.New(key)	

	if err != nil {
		log.Fatalf("can't init chacha20poly1305 cipher. %s", err)
	}

	token := cphr.Seal(nonce, nonce, buf, nil)
	return token
}

func NewChallange(id uint64) []byte {
	now := time.Now().Unix()
	msg := make([]byte, 64)	
	rand.Read(msg)

	// ID
	buf, err := binary.Append(nil, binary.BigEndian, id)
	if err != nil {
		log.Fatalf("can't write 'id' to buf. %s", err)
	}

	// Message
	buf = append(buf, msg...)

	// Time
	buf, err = binary.Append(buf, binary.BigEndian, now)
	if err != nil {
		log.Fatalf("can't write 'now' to buf. %s", err)
	}

	return buf
}

func GetAnswer(data *[]byte) (uint64, []byte, int64, error) {
	var tmsp int64
	var id uint64
	var msg []byte

	// Get id
	if len(*data) < 8 {
		err := errors.New("can't parse 'id' w/o 8 bytes")
		return id, msg, tmsp, err
	}
	id = binary.BigEndian.Uint64((*data)[:8])

	// Get messsage
	if len((*data)[8:]) < 64 {
		err := errors.New("can't parse 'msg' w/o 64 bytes")
		return id, msg, tmsp, err
	}
	msg = (*data)[8:72]

	// Get timestamp
	if len((*data)[72:]) < 8 {
		err := errors.New("can't parse 'tmsp' w/o 8 bytes")
		return id, msg, tmsp, err
	}
	tmsp = int64(binary.BigEndian.Uint64((*data)[72:80]))

	return id, msg, tmsp, nil
}

type NegotiatePacket struct {
	CId uint64
	Id uint64
	Method byte
	Token []byte			
	Challenge []byte   
	Answer []byte      
	Key []byte         
	Padding []byte     

	// | CID | ID | TOKEN_L | TOKEN | ENC_L | ENC (CHALL, ANS, KEY, RND) | PAD |
	Raw []byte     
}

func NewNegotiatePacket(
	cid, id uint64, 
	method byte, 
	token, challenge, answer, key, padding []byte,
	cphr *xcrypto.XCipher,
) NegotiatePacket {
	// Confirm the implemenation of binary.Append, below use case won't cause
	// error to be return, err can be safely igonred
	// Details: https://cs.opensource.google/go/go/+/refs/tags/go1.24.2:src/encoding/binary/binary.go;l=470
	raw, _ := binary.Append(nil, binary.BigEndian, cid)
	raw, _ = binary.Append(raw, binary.BigEndian, id)
	raw, _ = binary.Append(raw, binary.BigEndian, method)

	token_len     := uint32(len(token))	
	challenge_len := uint32(len(challenge))
	answer_len    := uint32(len(answer))
	key_len       := uint32(len(key))
	padding_len   := uint32(len(padding))

	plaintext, _ := binary.Append(nil, binary.BigEndian, challenge_len)
	plaintext, _  = binary.Append(plaintext, binary.BigEndian, answer_len)
	plaintext, _  = binary.Append(plaintext, binary.BigEndian, key_len)
	plaintext = append(plaintext, challenge...)
	plaintext = append(plaintext, answer...)
	plaintext = append(plaintext, key...)

	bytes := make([]byte, 32)
	rand.Read(bytes)	
	plaintext = append(plaintext, bytes...)

	ciphertext := cphr.Encrypt(&plaintext)
	ciphertext_len := uint32(len(ciphertext))

	raw, _ = binary.Append(raw, binary.BigEndian, token_len)
	if token_len != 0 {
		raw = append(raw, token...)
	}

	raw, _ = binary.Append(raw, binary.BigEndian, ciphertext_len)
	raw = append(raw, ciphertext...)

	raw, _ = binary.Append(raw, binary.BigEndian, padding_len)
	if padding_len != 0 {
		raw = append(raw, padding...)
	}

	return NegotiatePacket {
		cid,
		id,
		method,
		token,
		challenge,
		answer,
		key,
		padding,
		raw,	
	}
}

func NegotiatePacketFromBeBytes(
	data *[]byte, 
	cphr *xcrypto.XCipher,
) (NegotiatePacket, error) {
	var cid, id uint64
	var method byte
	var token, chall, ans, key, padding, raw []byte

	// CId + Id + Method + Token size 
	if len(*data) < (8 + 8 + 1 + 4) {
		err := errors.New("can't parse NegotiatePacket w/o enough bytes")
		return NegotiatePacket{}, err
	}

	// Get CId, Id, Method
	cid = binary.BigEndian.Uint64((*data)[0:8])
	id  = binary.BigEndian.Uint64((*data)[8:16])
	method = (*data)[16]

	// Get Token
	token_len := int(binary.BigEndian.Uint32((*data)[17:21]))
	if len((*data)[21:]) < token_len {
		err := errors.New("can't parse NegotiatePacket 'token_size' w/o enough bytes")
		return NegotiatePacket{}, err
	}
	token = (*data)[21:21+token_len]

	// Challange, Answer, Key are all in single encrypted message
	// Decrypt the encrypted segment first, then parse fields out
	start, end := 21+token_len, 21+token_len+4
	if len((*data)[start:]) < 4 {
		err := errors.New("can't parse NegotiatePacket 'cphrtxt_size' w/o enough bytes")
		return NegotiatePacket{}, err
	}
	cphrtxt_len := int(binary.BigEndian.Uint32((*data)[start:end]))

	start = end
	end += cphrtxt_len
	if len((*data)[start:]) < cphrtxt_len {
		err := errors.New("can't parse NegotiatePacket 'cphrtxt' w/o enough bytes")
		return NegotiatePacket{}, err
	}

	cphrtxt := (*data)[start:end]
	plntxt, err := cphr.Decrypt(&cphrtxt)
	if err != nil {
		return NegotiatePacket{}, err	
	}

	// Check if there enough bytes to parse the field size out
	if len(plntxt) < 12 {
		err := errors.New("can't parse 'chall', 'ans', 'key' out, not enough bytes")
		return NegotiatePacket{}, err
	}

	// Parse the size of chall, ans, key out
	chall_len := int(binary.BigEndian.Uint32(plntxt[0:4]))
	ans_len   := int(binary.BigEndian.Uint32(plntxt[4:8]))
	key_len   := int(binary.BigEndian.Uint32(plntxt[8:12]))

	// Check if there enough bytes to parse the values of chall, ans, key out
	// The last 32 bytes are random bytes
	if (len(plntxt[12:])-32) < chall_len+ans_len+key_len {
		err := errors.New("can't parse values of 'chall', 'ans', 'key' out, not enough bytes")
		return NegotiatePacket{}, err
	}

	if chall_len != 0 {
		chall = plntxt[12:12+chall_len]
	}

	if ans_len != 0 {
		ans = plntxt[12+chall_len:12+chall_len+ans_len]	
	}

	if key_len != 0 {
		key = plntxt[12+chall_len+ans_len:12+chall_len+ans_len+key_len]	
	}

	padding_start := 8 + 8 + 1 + (4+token_len) + (4+cphrtxt_len) 
	if len((*data)[padding_start:]) < 1200 {
		err := errors.New("can't parse 'padding', not enough bytes (1200 bytes)")
		return NegotiatePacket{}, err
	}
	padding = (*data)[padding_start+4:padding_start+4+1200]

	raw = *data

	return NegotiatePacket {
		cid,
		id,
		method,
		token,
		chall,
		ans,
		key,
		padding,
		raw,
	}, nil
}

func NewInit(cphr *xcrypto.XCipher) NegotiatePacket {	
	na := make([]byte, 0)
	padding := make([]byte, 1200)
	rand.Read(padding)

	return NewNegotiatePacket(0, 0, INIT, na, na, na ,na, padding, cphr)
}

func NewRetry(addr string, cphr *xcrypto.XCipher) NegotiatePacket {
	na := make([]byte, 0)
	token := NewToken(addr)
	padding := make([]byte, 1200)
	rand.Read(padding)

	return NewNegotiatePacket(0, 0, RETRY, token, na, na ,na, padding, cphr)
}

func NewInit2(
	id uint64, 
	token, chall []byte, 
	cphr *xcrypto.XCipher,
) NegotiatePacket {
	na := make([]byte, 0)
	padding := make([]byte, 1200)
	rand.Read(padding)
	
	return NewNegotiatePacket(0, id, INIT2, token, chall, na, na, padding, cphr)
}

func NewInitAck(
	cid, id uint64,
	chall, ans, key []byte,
	cphr *xcrypto.XCipher,
) NegotiatePacket {
	na := make([]byte, 0)	
	padding := make([]byte, 1200)
	rand.Read(padding)
	
	return NewNegotiatePacket(cid, id, INITACK, na, chall, ans, key, padding, cphr)
}

func NewInitDone(
	cid, id uint64,
	ans []byte,
	cphr *xcrypto.XCipher,
) NegotiatePacket {
	na := make([]byte, 0)	
	padding := make([]byte, 1200)
	rand.Read(padding)
	
	return NewNegotiatePacket(cid, id, INITDONE, na, na, ans, na, padding, cphr)
}
