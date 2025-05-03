package test

import (
    "log"
    "slices"
    "encoding/base64"
    "crypto/rand"
    "testing"

    "drill/pkg/x_crypto"
)

func TestXCipher(t *testing.T) {
    b_pkey := make([]byte, 32)
    rand.Read(b_pkey)

    pkey := base64.StdEncoding.EncodeToString(b_pkey)
    xcipher := x_crypto.NewXCipher(pkey)

    msg := []byte("Hello World! This is a test message for XCipher")

    ciphertext := xcipher.Encrypt(&msg)
    plaintext, err := xcipher.Decrypt(&ciphertext)

    if err != nil {
        log.Fatal(err)
    }

    if !slices.Equal(msg, plaintext) {
        log.Fatalf("Unmatched want and got. want: %v\ngot: %v\n", msg, plaintext) 
    }
}