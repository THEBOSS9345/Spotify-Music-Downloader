package app

import (
	"sync"
)

type Broker struct {
	mu      sync.RWMutex
	clients map[chan DownloadState]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		clients: make(map[chan DownloadState]struct{}),
	}
}

func (b *Broker) Subscribe() chan DownloadState {
	ch := make(chan DownloadState, 8)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *Broker) Unsubscribe(ch chan DownloadState) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *Broker) Publish(state DownloadState) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- state:
		default:
		}
	}
}

func (b *Broker) HasClients() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients) > 0
}
