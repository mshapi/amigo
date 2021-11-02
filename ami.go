package amigo

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	ErrEmptyDataOrTimedOut         = errors.New("empty response or timed out")
	ErrDefaultHandlerAlreadyExists = errors.New("default event handler already exists")
)

type AMI struct {
	mu sync.RWMutex

	ctx context.Context

	conn   net.Conn
	config *ConnectionConfig

	sender chan message

	defHandler EventHandler
	handlers   *amiHandler
}

func New(ctx context.Context, config *ConnectionConfig) (*AMI, error) {
	config.prepare()

	conn, err := config.getNetConn()
	if err != nil {
		return nil, err
	}

	res := &AMI{
		ctx:      ctx,
		conn:     conn,
		config:   config,
		sender:   make(chan message, config.SenderBufferSize),
		handlers: newAMIHandler(config.RequestTimeout),
	}

	res.setHandlers()

	if err := res.login(); err != nil {
		return nil, err
	}

	return res, nil
}

func (amiConn *AMI) login() error {
	res, err := amiConn.SendRequest(NewMessage(message{
		keyAction:   actionLogin,
		keyUsername: amiConn.config.Username,
		keyPassword: amiConn.config.Password,
	}))
	if err != nil {
		return err
	}

	if len(res) != 1 || res[0].Get("Response") != "Success" {
		var msg = "unknown"
		if len(res) == 1 {
			msg = res[0].Get("Message")
		}
		return fmt.Errorf("login error: %s", msg)
	}
	return nil
}

func (amiConn *AMI) Close() error {
	return amiConn.conn.Close()
}

func (amiConn *AMI) AsyncRequest(action *Message, handler EventHandler) {
	amiConn.handlers.Set(handlerTypeRequest, action.GetActionID(), handler)
	amiConn.sender <- action.fields
}

func (amiConn *AMI) AsyncRequestChan(action *Message) <-chan *Message {
	req := requestChan{
		closed:  false,
		channel: make(chan *Message, amiConn.config.SenderBufferSize),
	}
	amiConn.AsyncRequest(action, func(msg *Message) {
		go func(m *Message) {
			if m == nil {
				req.Close()
				return
			}
			req.Add(m)
		}(msg)
	})
	return req.channel
}

func (amiConn *AMI) SendRequest(action *Message) (res ActionResponse, err error) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	reqData := requestData{list: make(ActionResponse, 0)}

	amiConn.AsyncRequest(action, func(event *Message) {
		if event == nil {
			wg.Done()
			return
		}
		reqData.Add(event)
	})

	wg.Wait()

	if reqData.Len() == 0 {
		return nil, ErrEmptyDataOrTimedOut
	}

	return reqData.GetList(), err
}

func (amiConn *AMI) getDefaultEventHandler() EventHandler {
	amiConn.mu.RLock()
	defer amiConn.mu.RUnlock()
	return amiConn.defHandler
}

func (amiConn *AMI) SetDefaultEventHandler(h EventHandler) error {
	amiConn.mu.Lock()
	defer amiConn.mu.Unlock()
	if nil != amiConn.defHandler {
		return ErrDefaultHandlerAlreadyExists
	}
	amiConn.defHandler = h
	return nil
}

func (amiConn *AMI) SetEventHandler(eventType string, h EventHandler) error {
	if amiConn.handlers.Get(handlerTypeEvent, eventType) != nil {
		return fmt.Errorf("handler of event type '%s' already exists", eventType)
	}
	amiConn.handlers.Set(handlerTypeEvent, eventType, h)
	return nil
}

func (amiConn *AMI) setHandlers() {
	go func() {
		var msg message
		for {
			select {
			case <-amiConn.ctx.Done():
				return
			case tmp := <-amiConn.sender:
				msg = tmp
			case <-time.After(amiConn.config.KeepAliveTimeout):
				// Keep-Alive
				msg = message{
					keyAction: actionPing,
				}
			}
			// send request
			_, _ = amiConn.conn.Write(serialize(msg))
		}
	}()

	go func() {
		reader := bufio.NewReader(amiConn.conn)
		for {
			select {
			case <-amiConn.ctx.Done():
				_ = amiConn.conn.Close()
				return
			default:
				if m := readMessage(reader); m != nil {
					go amiConn.handleMessage(m)
				}
			}
		}
	}()
}

func (amiConn *AMI) handleMessage(msg *Message) {
	_ = amiConn.handlers.Handle(handlerTypeRequest, msg.GetActionID(), msg)

	defHandler := amiConn.getDefaultEventHandler()
	event := msg.GetEventType()
	useDefHandler := defHandler != nil

	if event != "" {
		err := amiConn.handlers.Handle(handlerTypeEvent, event, msg)
		useDefHandler = useDefHandler && err != nil
	}

	if useDefHandler {
		defHandler(msg)
	}
}

func readMessage(reader *bufio.Reader) *Message {
	var lines bytes.Buffer
	for {
		tmp, _, err := reader.ReadLine()
		if err != nil {
			return nil
		}

		lines.Write(tmp)
		if len(tmp) == 0 {
			return parseMessage(lines.Bytes())
		}

		lines.WriteString("\r\n")
	}
}
