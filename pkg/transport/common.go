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

func WriteAllUDPAddr(conn *net.UDPConn, data []byte, addr *net.UDPAddr) error {
	written := 0
	stop := len(data)

	for {
		n, err := conn.WriteToUDP(data[written:], addr)
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
	endpoints map[uint64] chan T
}

func NewChannelMap[T any]() *ChannelMap[T]{
	return &ChannelMap[T]{ 
		counter: 1,
		endpoints: make(map[uint64]chan T),
	}
}

func (cm *ChannelMap[T]) Create() (chan T, uint64){	
	ch := make(chan T, 65535)
	var id uint64

	cm.mu.Lock()
	cm.endpoints[cm.counter] = ch
	id = cm.counter
	cm.counter += 1
	cm.mu.Unlock()

	return ch, id
}

func (cm *ChannelMap[T]) Get(cid uint64) (chan T, bool) {
	cm.mu.Lock()
	ch, ok := cm.endpoints[cid]
	cm.mu.Unlock()
	return ch, ok
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
	cm.mu.RUnlock()	
	return fmt.Errorf("channel %d not found", id)
}