package transport

type Window struct {
	Buffer []byte		
	LastFrame Frame
	Sequence uint64
	Source uint64
	Destination uint64
}

func NewWindow(src, dst uint64) Window {
	buf := []byte{}	
	last := Frame{}
	seq := uint64(0)

	return Window {
		buf,
		last,
		seq,
		src,
		dst,
	}
}

func (win *Window) Push(data *[]byte) {
	win.Buffer = append(win.Buffer, (*data)...)
}

func (win *Window) Update() {
	win.Sequence += 1
}

func (win *Window) Next() Frame {
	return NewFrame(
		FWD, 
		win.Sequence,
		win.Source, 
		win.Destination,
		win.Buffer,
	)	
}