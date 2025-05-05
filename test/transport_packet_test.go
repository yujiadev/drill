package test

import (
    "time"
    "fmt"
    "log"
    "slices"
    "errors"
    "testing"
    "crypto/rand"

    "drill/pkg/xcrypto"
    "drill/pkg/transport"
)

const PKEY = "7abY7sBqNrtN5Z+NElo19hBDO1ixZ1+EGrrMq0gAjeE="

func TestNewChallange(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)

    challenge := transport.NewChallange(123456789)
    ciphertext := cphr.Encrypt(&challenge)
    plaintext, err1 := cphr.Decrypt(&ciphertext)
    if err1 != nil {
        log.Fatalf("Can't decrypt challenge. %s", err1)
    }

    //id, message, timestamp, err2 := transport.GetAnswer(&plaintext)
    id, _, _, err2 := transport.GetAnswer(&plaintext)
    if err2 != nil {
        log.Fatalf("can't get answer. %s", err2)
    }

    if !slices.Equal(challenge, plaintext) {
        log.Fatalf(
            "Unmatched answer (slice). want: %v\ngot: %v\n", 
            challenge, 
            plaintext,
        )
    }

    if 123456789 != id {
        log.Fatalf("Unmatched answer id. want: %v\ngot: %v\n", id, 123456789)
    }
}

func TestGetAnswer(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)

    challenge := transport.NewChallange(123456789)
    ciphertext := cphr.Encrypt(&challenge)
    plaintext, err1 := cphr.Decrypt(&ciphertext)
    if err1 != nil {
        log.Fatalf("Can't decrypt challenge. %s", err1)
    }

    //id, message, timestamp, err2 := transport.GetAnswer(&plaintext)
    id, message, timestamp, err2 := transport.GetAnswer(&plaintext)
    if err2 != nil {
        log.Fatalf("can't get answer. %s", err2)
    }

    // ID    
    if 123456789 != id {
        log.Fatalf("Unmatched answer id. want: %v\ngot: %v\n", id, 123456789)
    }

    // Message
    if !slices.Equal(message, challenge[8:72]) {
        log.Fatalf(
            "Unmatched answer message. want: %v\ngot: %v\n", 
            message, 
            challenge,
        )
    }

    if (timestamp+5) < time.Now().Unix() {
         log.Fatalf(
            "Unmatched answer timestamp. want: %v\ngot: %v\n", 
            message, 
            challenge,
        )
    }
}

func TestInitPacket(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)

    want := transport.NewInit(&cphr)
    bytes := want.Raw
    got, err := transport.NegotiatePacketFromBeBytes(&bytes, &cphr)

    if err != nil {
        log.Fatalf("NegotiatePacketFromBeBytes err (INIT). %s", err)
    }

    err = ComparePacketValue(
        got,
        0,
        0,
        transport.INIT,
        want.Token,
        want.Challenge,
        want.Answer,
        want.Key,
        want.Padding,
        want.Raw,
    )

    if err != nil {
        log.Fatalf("NegotiatePacket (INIT): %s", err)
    }
}

func TestRetryPacket(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)

    want := transport.NewRetry("127.0.0.1:8787", &cphr)
    bytes := want.Raw
    got, err := transport.NegotiatePacketFromBeBytes(&bytes, &cphr)

    if err != nil {
        log.Fatalf("NegotiatePacketFromBeBytes err (RETRY). %s", err)
    }

    err = ComparePacketValue(
        got,
        0,
        0,
        transport.RETRY,
        want.Token,
        want.Challenge,
        want.Answer,
        want.Key,
        want.Padding,
        want.Raw,
    )

    if err != nil {
        log.Fatalf("NegotiatePacket (RETRY): %s", err)
    }
}

func TestInit2Packet(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)

    id := uint64(987654321)
    token := transport.NewToken("127.0.0.1:8787")
    challenge := transport.NewChallange(1234567890)

    want := transport.NewInit2(id, token, challenge, &cphr)
    bytes := want.Raw
    got, err := transport.NegotiatePacketFromBeBytes(&bytes, &cphr)

    if err != nil {
        log.Fatalf("NegotiatePacketFromBeBytes err (INIT2). %s", err)
    }

    err = ComparePacketValue(
        got,
        0, 
        987654321,
        transport.INIT2,
        token,
        challenge,        
        []byte{},
        []byte{},
        want.Padding,
        want.Raw,
    )

    if err != nil {
        log.Fatalf("NegotiatePacket (INIT2): %s", err)
    }
}

func TestInitAckPacket(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)

    cid := uint64(13579)
    id := uint64(987654321)
    challenge := transport.NewChallange(1234567890)
    answer := make([]byte, 128)
    key := make([]byte, 128)
    rand.Read(answer)
    rand.Read(key)

    want := transport.NewInitAck(cid, id, challenge, answer, key, &cphr)
    bytes := want.Raw
    got, err := transport.NegotiatePacketFromBeBytes(&bytes, &cphr)

    if err != nil {
        log.Fatalf("NegotiatePacketFromBeBytes err (INITACK). %s", err)
    }

    err = ComparePacketValue(
        got,
        cid, 
        id,
        transport.INITACK,
        []byte{},
        challenge,        
        answer,
        key,
        want.Padding,
        want.Raw,
    )

    if err != nil {
        log.Fatalf("NegotiatePacket (INITACK): %s", err)
    }
}

func TestInitDonePacket(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)

    cid := uint64(13579)
    id := uint64(987654321)
    answer := make([]byte, 128)

    want := transport.NewInitDone(cid, id, answer, &cphr)
    bytes := want.Raw
    got, err := transport.NegotiatePacketFromBeBytes(&bytes, &cphr)

    if err != nil {
        log.Fatalf("NegotiatePacketFromBeBytes err (INITDONE). %s", err)
    }

    err = ComparePacketValue(
        got,
        cid, 
        id,
        transport.INITDONE,
        []byte{},
        []byte{},
        answer,
        []byte{},
        want.Padding,
        want.Raw,
    )

    if err != nil {
        log.Fatalf("NegotiatePacket (INITDONE): %s", err)
    }
}

func ComparePacketValue(
    got transport.NegotiatePacket, 
    cid, id uint64, 
    method byte, 
    token, challenge, answer, key, padding, raw []byte,
) error {
    // Compare CId
    if cid != got.CId {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.CId:\nwant: %v\ngot: %v\n",
            cid,
            got.CId,
        )
        return errors.New(msg)
    }

    // Compare Id
    if id != got.Id {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Id:\nwant: %v\ngot: %v\n",
            id,
            got.Id,
        )
        return errors.New(msg)
    }

    // Compare Method
    if method != got.Method {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Method:\nwant: %v\ngot: %v\n",
            method,
            got.Method,
        )
        return errors.New(msg)
    }

    // Compare Token
    if !slices.Equal(token, got.Token) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Token:\nwant: %v\ngot: %v\n",
            token,
            got.Token,
        )
        return errors.New(msg)
    }

    // Compare Challenge
    if !slices.Equal(challenge, got.Challenge) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Challenge:\nwant: %v\ngot: %v\n",
            challenge,
            got.Challenge,
        )
        return errors.New(msg)
    }

    // Compare Answer
    if !slices.Equal(answer, got.Answer) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Answer:\nwant: %v\ngot: %v\n",
            answer,
            got.Answer,
        )
        return errors.New(msg)
    }

    // Compare Key
    if !slices.Equal(key, got.Key) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Key:\nwant: %v\ngot: %v\n",
            key,
            got.Key,
        )
        return errors.New(msg)
    }

    // Compare Padding
    if !slices.Equal(padding, got.Padding) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Padding:\nwant: %v\ngot: %v\n",
            padding,
            got.Padding,
        )
        return errors.New(msg)
    }

    // Compare Raw
    if !slices.Equal(raw, got.Raw) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Raw:\nwant: %v\ngot: %v\n",
            raw,
            got.Raw,
        )
        return errors.New(msg)
    }

    return nil
}

func ComparePackets(got, want transport.NegotiatePacket) error {
    // Compare CId
    if want.CId != got.CId {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.CId:\nwant: %v\ngot: %v\n",
            want.CId,
            got.CId,
        )
        return errors.New(msg)
    }

    // Compare Id
    if want.Id != got.Id {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Id:\nwant: %v\ngot: %v\n",
            want.Id,
            got.Id,
        )
        return errors.New(msg)
    }

    // Compare Method
    if want.Method != got.Method {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Method:\nwant: %v\ngot: %v\n",
            want.Method,
            got.Method,
        )
        return errors.New(msg)
    }

    // Compare Token
    if !slices.Equal(want.Token, got.Token) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Token:\nwant: %v\ngot: %v\n",
            want.Token,
            got.Token,
        )
        return errors.New(msg)
    }

    // Compare Challenge
    if !slices.Equal(want.Challenge, got.Challenge) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Challenge:\nwant: %v\ngot: %v\n",
            want.Challenge,
            got.Challenge,
        )
        return errors.New(msg)
    }

    // Compare Answer
    if !slices.Equal(want.Answer, got.Answer) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Answer:\nwant: %v\ngot: %v\n",
            want.Answer,
            got.Answer,
        )
        return errors.New(msg)
    }

    // Compare Key
    if !slices.Equal(want.Key, got.Key) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Key:\nwant: %v\ngot: %v\n",
            want.Key,
            got.Key,
        )
        return errors.New(msg)
    }

    // Compare Padding
    if !slices.Equal(want.Padding, got.Padding) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Padding:\nwant: %v\ngot: %v\n",
            want.Padding,
            got.Padding,
        )
        return errors.New(msg)
    }

    // Compare Raw
    if !slices.Equal(want.Raw, got.Raw) {
        msg := fmt.Sprintf(
            "Unmatched NegotiatePacket.Raw:\nwant: %v\ngot: %v\n",
            want.Raw,
            got.Raw,
        )
        return errors.New(msg)
    }

    return nil
}