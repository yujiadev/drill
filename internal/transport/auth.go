package transport

import (
	"fmt"
	"bytes"
	"net"
	"time"
	"crypto/rand"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

func NewRetryToken(ip net.IP, secret []byte) []byte {
	token := make([]byte, 0, 20)

	// Time
	created := make([]byte, 4)
	binary.BigEndian.PutUint32(created, uint32(time.Now().Unix()))
	token = append(token, created...)

	// IP
	token = append(token, ip...)

	// Add HMAC to ensure message integrity and verification (anti-tamper)
	h := hmac.New(sha256.New, secret)
	h.Write(token)
	token = append(token, h.Sum(nil)...)

	return token
}

func ValidateRetryToken(token []byte, ip net.IP, secret []byte) bool {
	if len(token) < 4 + len(ip) + 32 {
		return false
	}

	// Verify time
	created := binary.BigEndian.Uint32(token[:4])
	createdTime := time.Unix(int64(created), 0)

	if time.Since(createdTime) > 2*time.Second {
    	return false
	}

	// Verify ip
	if !bytes.Equal(token[4:4+len(ip)], ip) {
		return false
	}

	// Verify HMAC 
	h := hmac.New(sha256.New, secret)
	h.Write(token[:4+len(ip)])
	expected := h.Sum(nil)

	return hmac.Equal(token[4+len(ip):], expected)
}

type AuthToken struct {
	Created 	time.Time
	Challenge 	[]byte
	Answer    	[]byte
	Key       	[]byte
}

func NewAuthToken(ans, key []byte) AuthToken {
	created := time.Now().Truncate(time.Second)
	challenge := make([]byte, 32)
	rand.Read(challenge)

	return AuthToken {
		created,
		challenge,
		ans,
		key,
	}
}

func (t *AuthToken) AsBytes() []byte {
	// time + challenge + answer + key
	token := make([]byte, 0, 8+32+32+32)	

	// Time
	created := make([]byte, 8)
	binary.BigEndian.PutUint64(created, uint64(t.Created.Unix()))
	token = append(token, created...)

	// Challenge
	token = append(token, t.Challenge...)

	// Answer
	token = append(token, t.Answer...)

	// Key
	token = append(token, t.Key...)

	return token
}

func ParseAuthToken(data []byte) (AuthToken, error) {
	// time + challenge + answer + key
	const EXACT int = 8 + 32 + 32 + 32	

	if len(data) != EXACT {
		return AuthToken{}, fmt.Errorf(
			"unmatched expected token length, want %v, got %v",
			EXACT,
			len(data),
		)
	}

	// Time
	created := time.Unix(int64(binary.BigEndian.Uint64(data[0:8])), 0)

	// Challenge
	challenge := make([]byte, 0, 32)
	challenge = append(challenge, data[8:40]...)

	// Answer
	answer := make([]byte, 0, 32)
	answer = append(answer, data[40:72]...)

	// Key
	key := make([]byte, 0, 32)
	key = append(key, data[72:104]...)

	return AuthToken {
		created,
		challenge,
		answer,
		key,
	}, nil
}

func (t *AuthToken) ValidateAuthToken(answer []byte) bool {	
	// Verify time
	if time.Since(t.Created) > 2*time.Second {
		return false
	}

	// Verify answer
	if !bytes.Equal(answer, t.Answer) {
		return false
	}

	return true
}