package amigo

import (
	"bufio"
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func createTestServerAndClient() net.Conn {
	server, client := net.Pipe()
	go func() {
		reader := bufio.NewReader(server)
		for {
			m := readMessage(reader)
			if m != nil {
				retMsg := m.Copy()
				switch m.GetAction() {
				case actionLogin:
					retMsg.Set("Response", "Success")
				case actionPing:
					retMsg.Set("Ping", "Pong")
				default:
					retMsg.Set("test", "test")
				}
				_, _ = server.Write(serialize(retMsg.GetFields()))
			}
		}
	}()
	return client
}

func createNewTestAMI(t *testing.T) *AMI {
	amiConn, err := New(context.Background(), &ConnectionConfig{
		Conn: createTestServerAndClient(),
	})
	if err != nil {
		t.Error(err)
		return nil
	}
	return amiConn
}

func TestNew(t *testing.T) {
	createNewTestAMI(t)
}

func TestAMI_SendRequest(t *testing.T) {
	amiConn := createNewTestAMI(t)
	res, err := amiConn.SendRequest(NewMessage(message{
		keyAction: "Test",
	}))
	if err != nil {
		t.Error(err)
	}
	if len(res) == 0 {
		t.Error(errors.New("return empty result"))
	}
}

func TestAMI_AsyncRequest(t *testing.T) {
	amiConn := createNewTestAMI(t)
	wait := make(chan struct{})
	amiConn.AsyncRequest(NewMessage(message{
		keyAction: "Ttt",
	}), func(action *Message) {
		if action == nil {
			return
		}
		wait <- struct{}{}
	})
	select {
	case <-time.After(amiConn.config.RequestTimeout):
		t.Error("callback not called")
	case <-wait:
		return
	}
}

func TestAMI_AsyncRequestChan(t *testing.T) {
	amiConn := createNewTestAMI(t)
	ch := amiConn.AsyncRequestChan(NewMessage(message{
		keyAction: "Ttt",
	}))
	counter := 0
	for range ch {
		counter++
	}
	if counter == 0 {
		t.Error("empty result in channel")
	}
}

func TestAMI_SetDefaultEventHandler(t *testing.T) {
	amiConn := createNewTestAMI(t)
	tmp := requestData{list: ActionResponse{}}
	err := amiConn.SetDefaultEventHandler(func(action *Message) {
		tmp.Add(action)
	})
	if err != nil {
		t.Error(err)
		return
	}
	_, _ = amiConn.SendRequest(NewMessage(message{keyEventType: "Test"}))
	if tmp.Len() == 0 {
		t.Error(errors.New("default event handler not working"))
	}
}

func TestAMI_SetEventHandler(t *testing.T) {
	amiConn := createNewTestAMI(t)
	tmp := requestData{list: ActionResponse{}}
	err := amiConn.SetEventHandler("Test", func(action *Message) {
		tmp.Add(action)
	})
	if err != nil {
		t.Error(err)
	}
	_, _ = amiConn.SendRequest(NewMessage(message{keyEventType: "Test"}))
	if tmp.Len() == 0 {
		t.Error(errors.New("event handler not working"))
	}
}
