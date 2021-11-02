package amigo

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

type EventHandler func(action *Message)

type handlerType uint8

const (
	handlerTypeEvent handlerType = iota
	handlerTypeRequest
)

var ErrHandlerNotFound = errors.New("handler not found")

func (t handlerType) getPrefix() string {
	return strconv.FormatUint(uint64(t), 10)
}

type amiHandler struct {
	mutex          sync.RWMutex
	requestTimeout time.Duration
	handlers       map[string]EventHandler
}

func newAMIHandler(timeout time.Duration) *amiHandler {
	return &amiHandler{
		requestTimeout: timeout,
		handlers:       make(map[string]EventHandler),
	}
}

func (h *amiHandler) getHandlerKey(hType handlerType, action string) string {
	return hType.getPrefix() + ":" + action
}

func (h *amiHandler) set(hType handlerType, action string, handler EventHandler) {
	h.mutex.Lock()
	h.handlers[h.getHandlerKey(hType, action)] = handler
	h.mutex.Unlock()
}

func (h *amiHandler) remove(hType handlerType, action string) {
	h.mutex.Lock()
	delete(h.handlers, h.getHandlerKey(hType, action))
	h.mutex.Unlock()
}

func (h *amiHandler) Get(hType handlerType, action string) EventHandler {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.handlers[h.getHandlerKey(hType, action)]
}

func (h *amiHandler) Set(hType handlerType, action string, handler EventHandler) {
	if handler == nil {
		return
	}
	h.set(hType, action, handler)
	if hType != handlerTypeRequest {
		return
	}
	// remove request handler by timeout
	go func() {
		<-time.After(h.requestTimeout)
		handler(nil) // last data always is nil
		h.remove(hType, action)
	}()
}

func (h *amiHandler) Handle(hType handlerType, action string, m *Message) error {
	handler := h.Get(hType, action)
	if handler != nil {
		handler(m)
		return nil
	}
	return ErrHandlerNotFound
}
