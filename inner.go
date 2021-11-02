package amigo

import "sync"

type requestData struct {
	mu   sync.RWMutex
	list ActionResponse
}

func (r *requestData) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.list)
}

func (r *requestData) Add(msg *Message) {
	r.mu.Lock()
	r.list = append(r.list, msg)
	r.mu.Unlock()
}

func (r *requestData) GetList() ActionResponse {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.list
}

type requestChan struct {
	mu      sync.RWMutex
	closed  bool
	channel chan *Message
}

func (r *requestChan) Close() {
	r.mu.Lock()
	if !r.closed {
		close(r.channel)
		r.closed = true
	}
	r.mu.Unlock()
}

func (r *requestChan) Add(m *Message) {
	r.mu.RLock()
	if !r.closed {
		r.channel <- m
	}
	r.mu.RUnlock()
}
