package x_crypto

import (
	//"log"
	"fmt"
	"crypto/cipher"
	"golang.org/x/crypto/chacha20poly1305"
)

// A wrapper of golang's chacha20poly1305
type XCipher struct {
	cipher cipher.AEAD
}

func (cphr *XCipher) new() {

}

func (cphr *XCipher) encrypt() {

}

func (cphr *XCipher) decrypt() {

}
