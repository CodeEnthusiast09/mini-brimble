package logstream

import "sync"

type Hub struct {
	mu      sync.RWMutex
	streams map[string]map[chan string]struct{}
}

func NewHub() *Hub {
	return &Hub{
		streams: make(map[string]map[chan string]struct{}),
	}
}

func (h *Hub) Subscribe(deploymentID string) chan string {
	ch := make(chan string, 32)

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.streams[deploymentID] == nil {
		h.streams[deploymentID] = make(map[chan string]struct{})
	}

	h.streams[deploymentID][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(deploymentID string, ch chan string) {
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

func (h *Hub) Broadcast(deploymentID string, message string) {
	h.mu.RLock()
	subscribers := h.streams[deploymentID]
	channels := make([]chan string, 0, len(subscribers))
	for ch := range subscribers {
		channels = append(channels, ch)
	}
	h.mu.RUnlock()

	for _, ch := range channels {
		select {
		case ch <- message:
		default:
		}
	}
}
