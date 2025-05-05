package transport

type Sessions struct {
	Connections map[uint64] string	
}

func NewSessions() Sessions {
	return Sessions { make(map[uint64]string) }
}

func (s *Sessions) IsConnectionExist(cid uint64) bool {
	addr := s.Connections[cid]

	if len(addr) == 0 {
		return false
	}

	return true
}

func (s *Sessions) Register(cid uint64, addr string) {
	s.Connections[cid]	= addr
}