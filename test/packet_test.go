package test

import (
	"fmt"
	"log"
	"time"
	"bytes"
	"crypto/rand"
	"testing"
	txp "drill/internal/transport"
)

func verifyPacket(
	got txp.Packet,
	cid uint64,
	method byte,
	created time.Time,
	seq, src, dst uint64,
	payload []byte,
) error {
	if cid != got.ConnId {
		return fmt.Errorf(
			"unmatched Packet.ConnId, want %v, got %v",
			cid,
			got.ConnId,
		)
	}

	if method != got.Method {
		return fmt.Errorf(
			"unmatched Packet.Method, want %v, got %v",
			method,
			got.Method,
		)
	}

	if !created.Equal(got.Created) {
		return fmt.Errorf(
			"unmatched Packet.Created, want %v, got %v",
			created,
			got.Created,
		)
	}

	if seq != got.Seq {
		return fmt.Errorf(
			"unmatched Packet.Seq, want %v, got %v",
			seq,
			got.Seq,
		)
	}

	if src != got.Src {
		return fmt.Errorf(
			"unmatched Packet.Src, want %v, got %v",
			src,
			got.Src,
		)	
	}

	if dst != got.Dst {
		return fmt.Errorf(
			"unmatched Packet.Dst, want %v, got %v",
			dst,
			got.Dst,
		)
	}

	if !bytes.Equal(payload, got.Payload) {
		return fmt.Errorf(
			"unmatched Packet.Payload, want %v, got %v",
			payload,
			got.Payload,
		)
	}

	return nil
}

func TestNewPacket(t *testing.T) {
	pkt := txp.NewPacket(123, txp.TUN, 456, 789, 101112, []byte("hello world!"))

	created := pkt.Created

	err := verifyPacket(
		pkt,
		123,
		txp.TUN,
		created,
		456,
		789,
		101112,
		[]byte("hello world!"),
	)

	// Everything should work out
	if err != nil {
		log.Fatalf("%s\n", err)
	}

	// Test the parsing of the Packet
	raw := pkt.AsBytes()

	parsedPacket, err := txp.ParsePacket(raw)

	if err != nil {
		log.Fatalf("%s\n", err)
	}

	err = verifyPacket(
		parsedPacket,
		123,
		txp.TUN,
		created,
		456,
		789,
		101112,
		[]byte("hello world!"),
	)

	if err != nil {
		log.Fatalf("%s\n", err)
	}
}

func TestNewTunPacket(t *testing.T) {
	tunPkt := txp.NewTunPacket()
	wantPayload := tunPkt.Payload
	wantCreated := tunPkt.Created

	if err := verifyPacket(
		tunPkt,
		0,
		txp.TUN,
		wantCreated,
		0,
		0,
		0,
		wantPayload,
	); err != nil {
		log.Fatalf("%s\n", err)
	}

	// Test the parsing of the Packet
	raw := tunPkt.AsBytes()

	parsedPacket, err := txp.ParsePacket(raw)

	if err != nil {
		log.Fatalf("%s\n", err)
	}

	if err := verifyPacket(
		parsedPacket,
		0,
		txp.TUN,
		wantCreated,
		0,
		0,
		0,
		wantPayload,
	); err != nil {
		log.Fatalf("%s\n", err)
	}
}

func TestNewRetryPacket(t *testing.T) {
	// Generate random token
	retryToken := make([]byte, 1024)
	rand.Read(retryToken)

	retryPkt := txp.NewRetryPacket(retryToken)
	wantPayload := retryPkt.Payload
	wantCreated := retryPkt.Created

	if err := verifyPacket(
		retryPkt,
		0,
		txp.TUN,
		wantCreated,
		0,
		0,
		0,
		wantPayload,
	); err != nil {
		log.Fatalf("%s\n", err)
	}

	// Test the parsing of the Packet
	raw := retryPkt.AsBytes()

	parsedPacket, err := txp.ParsePacket(raw)

	if err != nil {
		log.Fatalf("%s\n", err)
	}

	if err := verifyPacket(
		parsedPacket,
		0,
		txp.TUN,
		wantCreated,
		0,
		0,
		0,
		wantPayload,
	); err != nil {
		log.Fatalf("%s\n", err)
	}
}

func TestNewAuthPacket(t *testing.T) {
	// Generate random token
	authToken := make([]byte, 1024)
	rand.Read(authToken)

	authPkt := txp.NewAuthPacket(123, authToken)
	wantPayload := authPkt.Payload
	wantCreated := authPkt.Created

	if err := verifyPacket(
		authPkt,
		123,
		txp.TUNAUTH,
		wantCreated,
		1,
		0,
		0,
		wantPayload,
	); err != nil {
		log.Fatalf("%s\n", err)
	}

	// Test the parsing of the Packet
	raw := authPkt.AsBytes()

	parsedPacket, err := txp.ParsePacket(raw)

	if err != nil {
		log.Fatalf("%s\n", err)
	}

	if err := verifyPacket(
		parsedPacket,
		123,
		txp.TUNAUTH,
		wantCreated,
		1,
		0,
		0,
		wantPayload,
	); err != nil {
		log.Fatalf("%s\n", err)
	}
}





