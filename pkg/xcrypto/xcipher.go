package xcrypto

import (
	"log"
	"fmt"
	"crypto/rand"
	"crypto/cipher"
	aead "golang.org/x/crypto/chacha20poly1305"
)

func RandomKey(nbytes int) []byte {
	key := make([]byte, nbytes)
	rand.Read(key)
	return key
}

type XCipher struct {
	Cipher cipher.AEAD
}

func NewXCipher(key []byte) XCipher {
	if len(key)	!= aead.KeySize {
		log.Panicf(
			"chacha20poly1305 key size needs to be 32 bytes, got %v\n", 
			len(key),
		)
	}

	cphr, err := aead.New(key)
	if err != nil {
		log.Panicf("can't create chacha20poly1305 cipher, %s\n", err)
	}

	return XCipher { cphr }
}

func (cphr *XCipher) Encrypt(plntxt []byte) []byte {
	nonce := make(
		[]byte,
		aead.NonceSize, 
		aead.NonceSize + len(plntxt) + aead.Overhead,
	)

	if _, err := rand.Read(nonce); err != nil {
		log.Panicf("init nonce error, %s\n", err)
	}

	cphrtxt := cphr.Cipher.Seal(nonce, nonce, plntxt, nil)
	return cphrtxt	
}

func (cphr *XCipher) Decrypt(cphrtxt []byte) ([] byte, error) {
	if len(cphrtxt) < aead.NonceSize {
		return nil, fmt.Errorf("can't decrypt ciphertext, unmatched nonce size")
	}

	nonce, cphrtxt := cphrtxt[:aead.NonceSize], cphrtxt[aead.NonceSize:]
	plntxt, err := cphr.Cipher.Open(nil, nonce, cphrtxt, nil)

	if err != nil {
		return nil, err
	}

	return plntxt, nil
}
