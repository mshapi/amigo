package amigo

import (
	"bytes"
	"github.com/google/uuid"
	"strconv"
	"strings"
	"sync"
)

const (
	keyAction    = "Action"
	keyActionID  = "ActionID"
	keyEventType = "Event"
)

const (
	keyUsername = "Username"
	keyPassword = "Secret"
)

const (
	actionLogin = "Login"
	actionPing  = "Ping"
)

type message map[string]string

type ActionResponse []*Message

type Message struct {
	mu     sync.RWMutex
	fields message
}

func NewMessage(data map[string]string) *Message {
	res := &Message{}
	res.SetFields(data)
	return res
}

func generateActionID() string {
	return uuid.New().String()
}

func (m *Message) Set(key, value string) {
	m.mu.Lock()
	m.fields[key] = value
	m.mu.Unlock()
}

func (m *Message) SetFields(data map[string]string) {
	m.mu.Lock()
	m.fields = data
	m.mu.Unlock()
}

func (m *Message) get(key string) (v string, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok = m.fields[key]
	return v, ok
}

func (m *Message) Get(key string) string {
	v, _ := m.get(key)
	return v
}

func (m *Message) GetFields() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.fields.copy()
}

func (m *Message) GetEventType() string {
	return m.Get(keyEventType)
}

func (m *Message) GetAction() string {
	return m.Get(keyAction)
}

func (m *Message) GetActionID() string {
	id, ok := m.get(keyActionID)
	if !ok || id == "" {
		id = generateActionID()
		m.Set(keyActionID, id)
	}
	return id
}

func (m *Message) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return string(serialize(m.fields))
}

func (m *Message) Copy() *Message {
	m.mu.RLock()
	newMsg := Message{
		fields: m.fields.copy(),
	}
	m.mu.RUnlock()
	return &newMsg
}

func serialize(data message) []byte {
	var res bytes.Buffer
	for k, v := range data {
		res.WriteString(k)
		res.WriteString(": ")
		res.WriteString(v)
		res.WriteString("\r\n")
	}
	res.WriteString("\r\n")
	return res.Bytes()
}

func parseMessage(lines []byte) *Message {
	var res = message{}

	var str string
	var k string
	var v string
	var counter uint8 = 0

	for _, str = range strings.Split(string(lines), "\r\n") {
		str = strings.Trim(str, "\r\n")
		if str == "" {
			continue
		}

		i := strings.Index(str, ": ")
		if i >= 0 && i+2 <= len(str) {
			k = str[:i]
			v = str[i+2:]
		} else {
			k = strconv.FormatUint(uint64(counter), 10)
			v = str
			counter++
		}
		res[k] = v
	}
	return NewMessage(res)
}

func (src message) copy() message {
	cp := make(message)
	for k, v := range src {
		cp[k] = v
	}
	return cp
}
