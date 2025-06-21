package obfuscate

import (
	"fmt"
	"encoding/binary"
	"drill/pkg/xcrypto"
)

type Obfuscate interface {
	Encode(data []byte) []byte
	Decode(data []byte) ([]byte, error)
	SetPkey(pkey []byte) 
}
//
// Factory function to create obfuscators
//
func BuildObfuscator(name string, pkey []byte) Obfuscate {
	switch name {
	case "basic":
		return NewBasicObfuscator(pkey)
	default:
		return nil
	}	
}

//
// Simple Demo of implemenation of Obfuscate interface
//
type BasicObfuscator struct {
	Cipher xcrypto.XCipher
}

func NewBasicObfuscator(pkey []byte) *BasicObfuscator {
	return &BasicObfuscator{
		xcrypto.NewXCipher(pkey),
	}
}

func (bf *BasicObfuscator) Encode(data []byte) []byte {
	ciphertext := bf.Cipher.Encrypt(data)
	size := len(data)
	encoded := make([]byte, 0, 4+size)

	encoded, _ = binary.Append(encoded, binary.BigEndian, uint32(size))	
	encoded = append(encoded, ciphertext...)

	return encoded
}

func (bf *BasicObfuscator) Decode(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return []byte{}, fmt.Errorf(
			"malform encoded data, not enough bytes to parse size out",
		)	
	}

	size := int(binary.BigEndian.Uint32(data[0:4]))

	if len(data[4:]) < size {
		return []byte{}, fmt.Errorf(
			"malform encoded data, expect %v bytes payload, but got %v bytes",
			size,
			len(data[4:]),
		)			
	}

	plaintext, err := bf.Cipher.Decrypt(data[4:])

	if err != nil {
		return []byte{}, fmt.Errorf("malform encoded data. %s", err)
	}

	return plaintext, nil 
}

func (bf *BasicObfuscator) SetPkey(pkey []byte) {
	bf.Cipher = xcrypto.NewXCipher(pkey)
}
