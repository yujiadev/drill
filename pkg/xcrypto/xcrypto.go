package xcrypto

import (
	"log"
	"errors"
	"encoding/base64"
	"crypto/rand"
	"crypto/cipher"
	aead "golang.org/x/crypto/chacha20poly1305"
)

// A wrapper of golang's chacha20poly1305
type XCipher struct {
	Cipher cipher.AEAD
}

// Th key need to be base64 string
func NewXCipher(key_str string) XCipher {
	key, err := base64.StdEncoding.DecodeString(key_str)
	if err != nil {
		log.Fatalf("XCipher::new() decode base64 string error.", err)
	}

	if len(key) != aead.KeySize {
		log.Fatalf("XCipher::new() wrong size key size %v", len(key))
	}

	cphr, err := aead.New(key)
	if err != nil {
		log.Fatalf("XCipher::new() error. %v", err)
	}

	return XCipher{ cphr }
}

func NewXCipherFromBytes(key []byte) XCipher {
	if len(key) != aead.KeySize {
		log.Fatalf("XCipher::new() wrong size key size %v", len(key))
	}

	cphr, err := aead.New(key)
	if err != nil {
		log.Fatalf("XCipher::new() error. %v", err)
	}

	return XCipher{ cphr }
}

func (cphr *XCipher) Encrypt(plaintext *[]byte) []byte {	
	nonce := make(
		[]byte,
		aead.NonceSize, 
		aead.NonceSize + len(*plaintext) + aead.Overhead,
	)

	if _, err := rand.Read(nonce); err != nil {
		log.Fatalf("XCipher::Encrypt() error, %s", err)
	}

	ciphertext := cphr.Cipher.Seal(nonce, nonce, *plaintext, nil)
	return ciphertext
}

func (cphr *XCipher) Decrypt(encrypted *[]byte) ([] byte, error) {
	if len(*encrypted) < aead.NonceSize {
		return nil, errors.New("XCipher::Decrypt() ciphertext too short")
	}

	nonce, ciphertext := (*encrypted)[:aead.NonceSize], (*encrypted)[aead.NonceSize:]
	plaintext, err := cphr.Cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
