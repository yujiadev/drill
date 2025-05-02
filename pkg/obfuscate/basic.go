package obfuscate

import (
	"encoding/binary"
)

type BasicObfuscator struct {}

func New() BasicObfuscator {
	return BasicObfuscator{}
}

func (obfs *BasicObfuscator) Obfuscate(data *[]byte) {
	size := (uint32)len(*data)
	obfuscated := make([]byte, 4 + size)	

	binary.BigEndian.PutUint32(obfuscate[0:], size)
}

func (obfs *BasicObfuscator) Deobfuscate(data *[]byte) {

}