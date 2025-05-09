package test

import (
	"fmt"
	"slices"
	"crypto/rand"
	mrand "math/rand"	
	"testing"
	"log"

	txp "drill/pkg/transport"
)

func TestFrame(t *testing.T) {
	payload := make([]byte, mrand.Intn(65536))
	rand.Read(payload)

	wantFrame := txp.NewFrame(txp.FCONN, 11, 12, 13, payload)
	gotFrame, err  := txp.ParseFrame(&wantFrame.Raw)

	// Parsing issue, shouldn't happen here
	if err != nil {
		log.Fatalf("Err parse frame: %s\n", err)
	}

	// Values check
	err = compareFrameValues(
		gotFrame,
		wantFrame.Method,
		wantFrame.Time,
		wantFrame.Sequence,
		wantFrame.Source,
		wantFrame.Destination,
		wantFrame.Payload,
	)

	if err != nil {
		log.Fatalf("Err comparse frame againt values: %s\n", err)
	}
}

// Compare the frame fields againt values
func compareFrameValues(
	got txp.Frame,
	method byte,
	time int64,
	seq, src, dst uint64,
	payload []byte,
) error {
	if method != got.Method {
		return fmt.Errorf(
			"unmatched frame method: want '%v', got '%v'", 
			method, 
			got.Method,
		)
	}

	if time != got.Time {
		return fmt.Errorf(
			"unmatched frame time: want '%v', got '%v'",
			time, 
			got.Time,
		)
	}

	if seq != got.Sequence {
		return fmt.Errorf(
			"unmatched frame sequence: want '%v', got '%v'",
			seq, 
			got.Sequence,
		)
	}

	if src != got.Source {
		return fmt.Errorf(
			"unmatched frame source: want '%v', got '%v'",
			src, 
			got.Source,
		)
	}

	if dst != got.Destination {
		return fmt.Errorf(
			"unmatched frame destination: want '%v', got '%v'",
			dst, 
			got.Destination,
		)
	}

	if !slices.Equal(payload, got.Payload) {
		return fmt.Errorf(
			"unmatched frame payload: want '%v', got '%v'",
			payload, 
			got.Payload,
		)		
	}

	return nil
}

func compareFrames(want, got txp.Frame) error {
	if want.Method != got.Method {
		return fmt.Errorf(
			"unmatched frame method: want '%v', got '%v'", 
			want.Method, 
			got.Method,
		)
	}

	if want.Time != got.Time {
		return fmt.Errorf(
			"unmatched frame time: want '%v', got '%v'",
			want.Time, 
			got.Time,
		)
	}

	if want.Sequence != got.Sequence {
		return fmt.Errorf(
			"unmatched frame sequence: want '%v', got '%v'",
			want.Sequence, 
			got.Sequence,
		)
	}

	if want.Source != got.Source {
		return fmt.Errorf(
			"unmatched frame source: want '%v', got '%v'",
			want.Source, 
			got.Source,
		)
	}

	if want.Destination != got.Destination {
		return fmt.Errorf(
			"unmatched frame destination: want '%v', got '%v'",
			want.Destination, 
			got.Destination,
		)
	}

	if !slices.Equal(want.Payload, got.Payload) {
		return fmt.Errorf(
			"unmatched frame payload: want '%v', got '%v'",
			want.Payload, 
			got.Payload,
		)		
	}

	return nil
}