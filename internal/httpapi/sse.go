package httpapi

import (
	"sync"

	"github.com/alphabravo-oss/webhookie/internal/store"
)

type Hub struct {
	mu   sync.Mutex
	subs map[chan store.Event]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: map[chan store.Event]struct{}{}}
}

func (h *Hub) Publish(ev store.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (h *Hub) Subscribe() (<-chan store.Event, func()) {
	ch := make(chan store.Event, 16)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		if _, ok := h.subs[ch]; ok {
			delete(h.subs, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

func (h *Hub) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}
