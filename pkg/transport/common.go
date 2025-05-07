package transport

import (
	"fmt"
	"net"
	"sync"
)

func WriteAllUDP(conn *net.UDPConn, data []byte) error {
	written := 0
	stop := len(data)

	for {
		n, err := conn.Write(data[written:])
		if err != nil {
			return err
		}

		written += n
		if written == stop {
			break
		}
	}

	return nil
}

type ChannelMap[T any] struct {
	mu sync.RWMutex
	counter uint64
	endpoints map[uint64] chan<-T
}

func NewChannelMap[T any]() *ChannelMap[T]{
	return &ChannelMap[T]{ 
		counter: 1,
		endpoints: make(map[uint64]chan<-T),
	}
}

func (cm *ChannelMap[T]) Create() chan T{	
	ch := make(chan T, 65535)

	cm.mu.Lock()
	cm.endpoints[cm.counter] = ch
	cm.counter += 1
	cm.mu.Unlock()

	return ch 
}

func (cm *ChannelMap[T]) Delete(id uint64) {
	cm.mu.Lock()
	if ch, exist := cm.endpoints[id]; exist {
		close(ch)	
		delete(cm.endpoints, id)
	}
	cm.mu.Unlock()
}

func (cm *ChannelMap[T]) Send(id uint64, value *T) error {
	cm.mu.RLock()	
	if ch, exist := cm.endpoints[id]; exist {
		ch <- *value
		return nil
	}

	return fmt.Errorf("channel %d not found", id)
}