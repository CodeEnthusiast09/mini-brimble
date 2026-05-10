package logstream

import (
	"sync"
	"time"
)

type Event struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type Hub struct {
	mu      sync.RWMutex
	streams map[string]map[chan Event]struct{}
}

func NewHub() *Hub {
	return &Hub{
		streams: make(map[string]map[chan Event]struct{}),
	}
}

func (h *Hub) Subscribe(deploymentID string) chan Event {
	ch := make(chan Event, 32)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.streams[deploymentID] == nil {
		h.streams[deploymentID] = make(map[chan Event]struct{})
	}

	h.streams[deploymentID][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(deploymentID string, ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	subscribers, ok := h.streams[deploymentID]
	if !ok {
		return
	}

	if _, ok := subscribers[ch]; !ok {
		return
	}

	delete(subscribers, ch)
	close(ch)

	if len(subscribers) == 0 {
		delete(h.streams, deploymentID)
	}
}

func (h *Hub) Broadcast(deploymentID string, event Event) {
	h.mu.RLock()
	subscribers := h.streams[deploymentID]
	channels := make([]chan Event, 0, len(subscribers))
	for ch := range subscribers {
		channels = append(channels, ch)
	}
	h.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- event:
		default:
		}
	}
}
