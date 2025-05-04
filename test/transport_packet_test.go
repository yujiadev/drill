package test

/*
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

func TestRetryPacket(t *testing.T) {
    retry := transport.NewRetry("127.0.0.1:8787") 
    token := retry.Token

    retry_send := retry.ToBeBytes() 
    retry_recv, err := transport.RetryFromBeBytes(&retry_send)

    if err != nil {
        log.Fatalf("Parse Retry error. %s", err)
    }

    if !slices.Equal(retry_recv.Token, token) {
        log.Fatalf("want: %v\n, got: %v\n", token, retry_recv.Token)
    }
}
*/

import (
    //"log"
    //"slices"
    //"encoding/base64"
    //"crypto/rand"
    "testing"

    "drill/pkg/transport"
)

func TestInitPacket(t *testing.T) {
    transport.NewToken("127.0.0.1:8787")
}
