package test

import (
	//"fmt"
	"net"
	"log"
	//"time"
	//"bytes"
	"crypto/rand"
	"testing"
	txp "drill/internal/transport"
)

func TestRetryToken(t *testing.T) {
	ip := net.ParseIP("192.168.1.11")
	secret := make([]byte, 32)
	rand.Read(secret)

	token := txp.NewRetryToken(ip, secret)

	if ok := txp.ValidateRetryToken(token, ip, secret); !ok {
		log.Fatalf("can't validate retry token")
	}

	wrongIP := net.ParseIP("192.168.1.12")
	if mustFail := txp.ValidateRetryToken(token, wrongIP, secret); mustFail {
		log.Fatalf("validation of retry token should fail b/c wrong IP")
	}

	wrongToken := make([]byte, 0, 20)
	wrongToken = append(wrongToken, token...)
	wrongToken[0] = 25
	wrongToken[1] = 25

	if mustFail := txp.ValidateRetryToken(wrongToken, ip, secret); mustFail {
		log.Fatalf("validation of retry token should fail b/c wrong token")
	}
}

func TestAuthToken(t *testing.T) {
	ip := net.ParseIP("192.168.1.11")
	ans := make([]byte, 32)
	key := make([]byte, 32)

	rand.Read(ans)
	rand.Read(key)

	authToken := txp.NewAuthToken(ip, ans, key)
	raw := authToken.AsBytes()
	parsedToken, err := txp.ParseAuthToken(raw)

	if err != nil {
		log.Fatalf("can't parse AuthToken. %v", err)
	}

	if ok := parsedToken.ValidateAuthToken(ip, ans); !ok {
		log.Fatalf("can't validate a ok AuthToken")
	}

	wrongAns := make([]byte, 32)
	rand.Read(wrongAns)

	if ok := authToken.ValidateAuthToken(ip, wrongAns); ok {
		log.Fatalf("validation of auth token should fail b/c wrong answer")
	}
	
	wrongIP := net.ParseIP("192.168.1.12")
	if ok := authToken.ValidateAuthToken(wrongIP, ans); ok {
		log.Fatalf("validation of auth token should fail b/c wrong IP")
	}
}
	