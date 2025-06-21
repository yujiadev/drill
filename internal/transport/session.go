package transport

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

type Sessions struct {
	mu 		sync.RWMutex
	counter atomic.Uint64
	sessions map[string]chan []byte
}

func NewSessions() *Sessions {
	ss := &Sessions {
		sessions: make(map[string]chan []byte),
	}

	ss.counter.Store(1)

	return ss
}

func (ss *Sessions) Create(addr *net.UDPAddr) (chan []byte, uint64) {
	ch := make(chan []byte, 65535)
	key := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)

	ss.mu.Lock()
	defer ss.mu.Unlock()

	cid := ss.counter.Load()
	ss.counter.Add(1)
	ss.sessions[key] = ch

	return ch, cid
}

func (ss *Sessions) Delete(addr *net.UDPAddr) {
	key := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)

	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ch, exists := ss.sessions[key]; exists {
		close(ch)
		delete(ss.sessions, key)
	}
}

func (ss *Sessions) Get(addr *net.UDPAddr) (chan []byte, bool) {
	key := fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)

	ss.mu.Lock()
	defer ss.mu.Unlock()

	ch, ok := ss.sessions[key]
	
	return ch, ok
}

type Endpoints struct {
	mu 		sync.RWMutex
	counter atomic.Uint64
	endpoints map[uint64]chan Packet
}

func NewEndpoints() *Endpoints {
	ep := &Endpoints {
		endpoints: make(map[uint64]chan Packet),
	}

	ep.counter.Store(1)

	return ep
}

func (ep *Endpoints) Create() (chan Packet, uint64) {
	ch := make(chan Packet, 65535)

	ep.mu.Lock()
	defer ep.mu.Unlock()

	id := ep.counter.Load()
	ep.counter.Add(1)
	ep.endpoints[id] = ch

	return ch, id
}

func (ep *Endpoints) Delete(id uint64) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ch, exists := ep.endpoints[id]; exists {
		close(ch)
		delete(ep.endpoints, id)
	}
}

func (ep *Endpoints) Get(id uint64) (chan Packet, bool) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	ch, ok := ep.endpoints[id]
	
	return ch, ok
}