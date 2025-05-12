package transport

import (
	"sync"
	"sync/atomic"
)

type ChannelMap[T any] struct {
	mu sync.RWMutex
	counter atomic.Uint64
	endpoints map[uint64] chan T
}

func NewChannelMap[T any]() *ChannelMap[T]{
	var counter atomic.Uint64
	counter.Add(1)

	return &ChannelMap[T]{ 
		counter: counter,
		endpoints: make(map[uint64]chan T),
	}
}

func (cm *ChannelMap[T]) Create() (chan T, uint64){	
	ch := make(chan T, 65535)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	id := uint64(cm.counter.Load())
	cm.counter.Add(1)

	cm.endpoints[id] = ch

	return ch, id
}

func (cm *ChannelMap[T]) Get(cid uint64) (chan T, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	ch, ok := cm.endpoints[cid]
	return ch, ok
}

func (cm *ChannelMap[T]) Delete(id uint64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if ch, exist := cm.endpoints[id]; exist {
		close(ch)	
		delete(cm.endpoints, id)
	}
}
