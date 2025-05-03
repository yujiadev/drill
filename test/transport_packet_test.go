package test

import (
    "log"
    "slices"
    //"encoding/base64"
    //"crypto/rand"
    "testing"

    "drill/pkg/transport"
)

func TestInitPacket(t *testing.T) {
    init := transport.NewInit()
    init_padding := init.Padding

    init_send := init.ToBeBytes()
    init_recv, err := transport.InitFromBeBytes(&init_send)

    if err != nil {
        log.Fatalf("Parse Init error. %s", err)
    }

    if !slices.Equal(init_recv.Padding, init_padding) {
        log.Fatalf("want: %v\n, got: %v\n", init_recv.Padding, init_padding)
    }
}