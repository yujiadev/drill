package obfuscate

import (
	"errors"
	"encoding/binary"
)

type BasicObfuscator struct {}

func NewBasicObfuscator() BasicObfuscator {
	return BasicObfuscator{}
}

func (obfs *BasicObfuscator) Obfuscate(data *[]byte) []byte {
	size := uint32(len(*data))
	buf := make([]byte, 4)	
	binary.BigEndian.PutUint32(buf, size)
	return append(buf, (*data)...)
}

func (obfs *BasicObfuscator) Deobfuscate(data *[]byte) ([]byte, error) {
	if len(*data) < 4 {
		return nil, errors.New("Not enough bytes to parse payload size out")
	}

	size := int(binary.BigEndian.Uint32((*data)[:4]))
	if len((*data)[4:]) < size {
		return nil, errors.New("Not enough bytes to parse payload out")
	}

	return (*data)[4: (4+size)], nil
}