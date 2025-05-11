package test

import (
    //"fmt"
    "log"
    //"slices"
    "testing"
    "crypto/rand"
    "time"
    mrand "math/rand"

    //"drill/pkg/xcrypto"
    txp "drill/pkg/transport"
)

func TestRecvPacer(t *testing.T) {
    pacer := txp.NewRecvPacer()

    mrand.Seed(time.Now().UnixNano())
    permutation := mrand.Perm(mrand.Intn(65535))

    size := 0
    for _, seq := range permutation {
        frame := txp.NewFrame(txp.FFWD, uint64(seq), 0, 0, []byte("test"))
        pacer.PushFrame(frame)
        size += 1
    }

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

func TestSendRecv(t *testing.T) {
    pacer := txp.NewSendPacer(12, 13) 

    data := make([]byte, 10485760) // 1 mb of random bytes
    rand.Read(data)
    pacer.PushBuffer(data)

    //
    // First Pop
    //
    base := 0
    npop := mrand.Intn(64)
    sentAcks := []uint64{}
    permutation := mrand.Perm(npop)

    // Pop  
    for j := 0; j <= npop; j ++ {
        _, ok := pacer.PopFrame()
        if !ok {
            log.Fatalf("Err pop frame (1st): (prep) pop is not unavailable\n")
        }
    }

    // Generate a random permutation of acks
    for _, seq := range permutation {
        sentAcks = append(sentAcks, uint64(base+seq))
    }

    // Sync the generated acks with pacer
    for _, ack := range sentAcks {
        pacer.RecvAck(ack)
    }

    _, ok := pacer.PopFrame()
    if !ok {
        log.Fatalf("Err pop frame (1st): (after) pop is not unavailable\n")
    }

    /*
    if frame.Sequence != uint64(base+npop+1) {
        log.Fatalf(
            "Err pop frame (1st): frame seq unmatched, want '%v', got '%v'\n",
            base+npop+1,
            frame.Sequence,
        )
    }
    */

    //
    // Second Pop
    //

    /*
    base = base + npop + 1
    npop = mrand.Intn(64)
    sentAcks = []uint64{}
    permutation = mrand.Perm(npop)

    // Pop  
    for j := 0; j <= npop; j ++ {
        _, ok := pacer.PopFrame()
        if !ok {
            log.Fatalf("Err pop frame (2rd): (prep) pop is not unavailable\n")
        }
    }

    // Generate a random permutation of acks
    for _, seq := range permutation {
        sentAcks = append(sentAcks, uint64(base+seq))
    }

    // Sync the generated acks with pacer
    for _, ack := range sentAcks {
        pacer.RecvAck(ack)
    }

    _, ok = pacer.PopFrame()
    if !ok {
        log.Fatalf("Err pop frame (2rd): (after) pop is not unavailable\n")
    }
    */

    /*
    if frame.Sequence != uint64(base+npop+2) {
        log.Fatalf(
            "Err pop frame (2rd): frame seq unmatched, want '%v', got '%v'\n",
            base+npop+2,
            frame.Sequence,
        )
    }
    */

}