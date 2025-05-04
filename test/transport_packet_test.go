package test

import (
    "log"
    "slices"
    "testing"

    "drill/pkg/xcrypto"
    "drill/pkg/transport"
)

const PKEY = "7abY7sBqNrtN5Z+NElo19hBDO1ixZ1+EGrrMq0gAjeE="

func TestNewChallange(t *testing.T) {

}

func TestGetAnswer(t *testing.T) {

}

func TestInitPacket(t *testing.T) {
    cphr := xcrypto.NewXCipher(PKEY)
    init := transport.NewInit(&cphr)
    bytes := init.Raw
    output_init, err := transport.NegotiatePacketFromBeBytes(&bytes, &cphr)

    if err != nil {
        log.Fatalf("NegotiatePacketFromBeBytes err (INIT). %s", err)
    }

    if !slices.Equal(init.Padding, output_init.Padding) {
        log.Fatalf(
            "Unmatched NegotiatePacket.Padding:\nwant: %v\ngot: %v\n",
            init.Padding,
            output_init.Padding,
        )
    }


    if !slices.Equal(init.Raw, output_init.Raw) {
        log.Fatalf(
            "Unmatched NegotiatePacket.Raw:\nwant: %v\ngot: %v\n",
            init.Raw,
            output_init.Raw,
        )
    }
}

func TestRetryPacket(t *testing.T) {
}

func TestInit2Packet(t *testing.T) {

}

func TestInitAckPacket(t *testing.T) {

}

func TestInitDonePacket(t *testing.T) {

}
