package transport

const (
	CONN byte = iota
	DISCONN
	FWD
	ACK
)

type Frame struct {
	Method byte
	Time int64
	Sequence uint64
	Source uint64
	Destination uint64
	Payload []byte
	Raw []byte
}

func NewFrame(method byte, seq, src, dst uint64, payload []byte) {

}

func ParseFrame(data *[]byte) (Frame, error) {

	return Frame{}, nil
}