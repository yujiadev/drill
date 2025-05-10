package test

import (
    //"fmt"
    "log"
    //"slices"
    "testing"
    //"crypto/rand"
    "time"
    mrand "math/rand"

    //"drill/pkg/xcrypto"
    txp "drill/pkg/transport"
)

func TestRecvPacerPushAndPop(t *testing.T) {
    pacer := txp.NewRecvPacer()

    mrand.Seed(time.Now().UnixNano())
    permutation := mrand.Perm(mrand.Intn(65535))

    size := 0
    for _, seq := range permutation {
        frame := txp.NewFrame(txp.FFWD, uint64(seq), 0, 0, []byte("test"))
        pacer.PushFrame(frame)
        size += 1
    }

    log.Println(size)

    for i := 0; i < size; i++ {
        frame, ok := pacer.PopFrame()

        if !ok {
            log.Fatalf("Err min seq frame should be popped\n")
        }

        if frame.Sequence != uint64(i) {
            log.Fatalf(
                "Err min seq frame should be popped. want: '%v', got: '%v'\n",
                i,
                frame.Sequence,
            )
        }
    }
}