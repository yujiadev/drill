package test

import (
    "fmt"
    "log"
    //"slices"
    "testing"
    //"crypto/rand"
    "time"
    "math/rand"

    //"drill/pkg/xcrypto"
    txp "drill/pkg/transport"
)

func TestRecvPacerPush(t *testing.T) {
    pacer := txp.NewRecvPacer()

    rand.Seed(time.Now().UnixNano())
    permutation := rand.Perm(1000)

    for _, value := range permutation {
        pacer.Push(uint64(value))
    }

    //fmt.Println(pacer.Sequence)

    output := []uint64{}
    size := len(pacer.Sequence)
    fmt.Printf("size is %v\n", size)

    for i := 0; i < size; i++ {
        seq, _ := pacer.Pop()
        output = append(output, seq)        
    }

    //fmt.Println(output)

    for i := 0; i < size; i++ {
        if uint64(i) != output[i] {
            log.Fatalf(
                "Unmatched: want '%v', got '%v', output '%v'", 
                i, 
                output[i],
                output,
            )
        }
    }
}