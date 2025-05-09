package test

import (
    //"time"
    "fmt"
    "log"
    "slices"
    "testing"
    "crypto/rand"

    "drill/pkg/xcrypto"
    txp "drill/pkg/transport"
)

const PKEY = "7abY7sBqNrtN5Z+NElo19hBDO1ixZ1+EGrrMq0gAjeE="

func TestNegotiateAtInit2(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)
    na := []byte{}

    // INIT2
    want := txp.NewNegotiate(12, na, na, &cphr)
    got, err := txp.ParseNegotiate(want.Raw, &cphr)

    if err != nil {
        log.Fatalf("Err parse negotiate: %s\n", err)
    }

    err = compareNegotiateValues(
        got,
        want.Time,
        want.Id,
        want.Challenge,
        want.Answer, 
        want.Key,
    )

    if err != nil {
        log.Fatalf("Err compare negotiate against values: %s\n", err)
    }
}

func TestNegotiateAtInitAck(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)
    init2 := txp.NewNegotiate(12, []byte{}, []byte{}, &cphr)

    ans := init2.Challenge
    key := make([]byte, 32)
    rand.Read(key)

    // INITACK
    want := txp.NewNegotiate(12, ans, key, &cphr) 
    got, err := txp.ParseNegotiate(want.Raw, &cphr)

    if err != nil {
        log.Fatalf("Err parse negotiate: %s\n", err)
    }

    err = compareNegotiateValues(
        got,
        want.Time,
        want.Id,
        want.Challenge,
        want.Answer, 
        want.Key,
    )

    if err != nil {
        log.Fatalf("Err compare negotiate against values: %s\n", err)
    }
}

func TestNegotiateAtInitDone(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)
    
    ans := make([]byte, 64)
    rand.Read(ans)

    // INITDONE
    want := txp.NewNegotiate(12, ans, []byte{}, &cphr) 
    got, err := txp.ParseNegotiate(want.Raw, &cphr)

    if err != nil {
        log.Fatalf("Err parse negotiate: %s\n", err)
    }

    err = compareNegotiateValues(
        got,
        want.Time,
        want.Id,
        want.Challenge,
        want.Answer, 
        want.Key,
    )

    if err != nil {
        log.Fatalf("Err compare negotiate against values: %s\n", err)
    }
}

func compareNegotiateValues(
    got txp.Negotiate,
    time int64,
    id uint64,
    chall, ans, key []byte,
) error {
    if time != got.Time {
        return fmt.Errorf(
            "unmatched negotiate time: want '%v', got '%v'\n", 
            time,
            got.Time,
        )
    }

    if id != got.Id {
        return fmt.Errorf(
            "unmatched negotiate id: want '%v', got '%v'\n",
            id,
            got.Id,
        )
    }

    if !slices.Equal(chall, got.Challenge) {
        return fmt.Errorf(
            "unmatched negotiate challenge: want '%v', got '%v'\n",
            chall,
            got.Challenge,
        )
    }

    if !slices.Equal(ans, got.Answer) {
        return fmt.Errorf(
            "unmatched negotiate challenge: want '%v', got '%v'\n",
            ans,
            got.Answer,
        )
    }

    if !slices.Equal(key, got.Key) {
        return fmt.Errorf(
            "unmatched negotiate challenge: want '%v', got '%v'\n",
            key,
            got.Key,
        )
    }

    return nil
}

func compareNegotiate(want, got txp.Negotiate) error {
    if want.Time != got.Time {
        return fmt.Errorf(
            "unmatched negotiate time: want '%v', got '%v'\n", 
            want.Time,
            got.Time,
        )
    }

    if want.Id != got.Id {
        return fmt.Errorf(
            "unmatched negotiate id: want '%v', got '%v'\n",
            want.Id,
            got.Id,
        )
    }

    if !slices.Equal(want.Challenge, got.Challenge) {
        return fmt.Errorf(
            "unmatched negotiate challenge: want '%v', got '%v'\n",
            want.Challenge,
            got.Challenge,
        )
    }

    if !slices.Equal(want.Answer, got.Answer) {
        return fmt.Errorf(
            "unmatched negotiate answer: want '%v', got '%v'\n",
            want.Challenge,
            got.Answer,
        )
    }

    if !slices.Equal(want.Key, got.Key) {
        return fmt.Errorf(
            "unmatched negotiate key: want '%v', got '%v'\n",
            want.Key,
            got.Key,
        )
    }

    return nil   
}