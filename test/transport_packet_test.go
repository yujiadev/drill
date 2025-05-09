package test

import (
    "fmt"
    "log"
    "slices"
    "crypto/rand"
    mrand "math/rand"   
    "testing"

    "drill/pkg/xcrypto"
    txp "drill/pkg/transport"
)

const PKEY2 = "7abY7sBqNrtN5Z+NElo19hBDO1ixZ1+EGrrMq0gAjeE="

func TestInitPacket(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)

    // INIT
    want := txp.NewInit(&cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (INIT): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }
}

func TestRetryPacket(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)

    // RETRY
    want := txp.NewRetry(12, []byte("127.0.0.1:8787"), &cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (RETRY): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }
}

func TestInit2Packet(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)
    token := make([]byte, mrand.Intn(65536))
    rand.Read(token)

    // INIT2
    want := txp.NewInit2(12, 13, token, &cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (INIT2): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }
}

func TestInitAck(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)
    ans := make([]byte, 64)
    key := make([]byte, 32)
    rand.Read(ans)
    rand.Read(key)

    // INITACK
    want := txp.NewInitAck(12, 13, ans, key, &cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (INITACK): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }
}

func TestInitDone(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)
    ans := make([]byte, 64)
    rand.Read(ans)

    // INITDONE
    want := txp.NewInitDone(12, 13, ans, &cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (INITDONE): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }   
}

func TestTx(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)
    data := make([]byte, mrand.Intn(65536))
    rand.Read(data)
    frame := txp.NewFrame(txp.FFWD, 12, 13, 14, data)

    // TX
    want := txp.NewTx(12, 13, frame, &cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (TX): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }    
}

func TestFin(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)

    // FIN
    want := txp.NewFin(12, 13, &cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (FIN): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }    
}

func TestFinAck(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY2)
    data := make([]byte, mrand.Intn(65536))
    rand.Read(data)

    // FINACK
    want := txp.NewFinAck(12, 13, &cphr)

    bytes := want.Raw
    got, err := txp.ParsePacket(bytes, &cphr)

    if err != nil {
        log.Fatalf("Err parse packet (FINACK): %s\n", err)
    }

    err = comparePackets(want, got)
    if err != nil {
        log.Fatalf("Err compare packets: %s\n", err)
    }    
}

func comparePackets(want, got txp.Packet) error {
    if want.ConnectionId != got.ConnectionId {
        return fmt.Errorf(
            "unmatched packet connection id: want '%v', got '%v'",
            want.ConnectionId,
            got.ConnectionId,
        )
    }

    if want.Id != got.Id {
        return fmt.Errorf(
            "unmatched packet id: want '%v', got '%v'",
            want.Id,
            got.Id,
        )
    }

    if want.Method != got.Method {
        return fmt.Errorf(
            "unmatched packet method: want '%v', got '%v'",
            want.Method,
            got.Method,
        )
    }

    if !slices.Equal(want.Token, got.Token) {
         return fmt.Errorf(
            "unmatched packet token: want '%v', got '%v'",
            want.Token,
            got.Token,
        )
    }

    if err := compareNegotiate(want.Authenticate, got.Authenticate); err != nil {
        return err
    }

    if err := compareFrames(want.Payload, got.Payload); err != nil {
        return fmt.Errorf(
            "unmatched packet payload: want '%v', got '%v'",
            want.Payload,
            got.Payload,
        ) 
    }

    return nil
}

