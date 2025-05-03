package test

import (
    "log"
    "slices"
    "testing"

    "drill/pkg/obfuscate"
)

func TestBasicObfuscator(t *testing.T) {
    msg := []byte("Hello World! This is a test message for BasicObfuscator")
    obfs := obfuscate.NewBasicObfuscator()

    obfs_msg := obfs.Obfuscate(&msg)
    deobfs_msg, err := obfs.Deobfuscate(&obfs_msg)

    if err != nil {
        log.Fatalf("BasicObfuscator::Deobfuscate error %s", err)
    }

    if !slices.Equal(msg, deobfs_msg) {
        log.Fatalf("Unmatched result\nwant: \n%v\ngot: \n%v\n", msg, deobfs_msg)
    }
}