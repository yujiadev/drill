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

func TestSendPacerPopFrameAndRecvAck(t *testing.T) {
    pacer := txp.NewSendPacer(12, 13) 
    data := make([]byte, 1048576) // 1 mb of random bytes
    pacer.PushBuffer(data)

    base := uint64(0)
    npop := mrand.Intn(63) + 1  // avoid 0
    sentAcks := []uint64{}
    permutation := mrand.Perm(npop)

    for i := 0; i < 21; i++ {
        // Pop  
        for j := 0; j < npop; j ++ {
            _, ok := pacer.PopFrame()
            if !ok {
                pacer.Report()
                log.Fatalf(
                    "Err pop frame: pop is not unavailable. "+
                    "iter: %v, base: %v, npop: %v\n", 
                    i,
                    base,
                    npop,
                )
            }
        }

        // Generate a random permutation of acks for the frames just popped
        for _, seq := range permutation {
            sentAcks = append(sentAcks, uint64(base)+uint64(seq))
        }

        // Sync the generated acks with pacer
        for _, ack := range sentAcks {
            pacer.RecvAck(ack)
        }

        if pacer.GetBacklog() != 0 {
            pacer.Report()
            log.Fatalf(
                "Err non empty backlog, want 0, got %v\n", 
                pacer.GetBacklog(),
            )
        }

        // Config for next round of testing, move parameter forward
        base = base + uint64(npop)
        npop = mrand.Intn(63) + 1  // avoid 0
        sentAcks = []uint64{}
        permutation = mrand.Perm(npop)
    }   
}