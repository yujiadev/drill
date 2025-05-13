package transport

import (
	"sync"
	"sync/atomic"
)

type ChannelMap[T any] struct {
	mu        sync.RWMutex
	counter   atomic.Uint64
	endpoints map[uint64]chan T
}

func NewChannelMap[T any]() *ChannelMap[T] {
	cm := &ChannelMap[T]{
		endpoints: make(map[uint64]chan T),
	}
	cm.counter.Store(1) // Initialize counter starting at 1
	return cm
}

func (cm *ChannelMap[T]) Create() (chan T, uint64) {
	ch := make(chan T, 65535)

	cm.mu.Lock()
	defer cm.mu.Unlock()

	id := cm.counter.Load()
	cm.counter.Add(1)

	cm.endpoints[id] = ch

	return ch, id
}

func (cm *ChannelMap[T]) Get(id uint64) (chan T, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	ch, ok := cm.endpoints[id]
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