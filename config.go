package amigo

import (
	"net"
	"strconv"
	"time"
)

const (
	defaultSenderBufferSize = 10
	defaultKeepAliveTimeout = 3 * time.Second
	defaultRequestTimeout   = 500 * time.Millisecond
)

type ConnectionConfig struct {
	Conn net.Conn

	Host string
	Port uint64

	Username string
	Password string

	SenderBufferSize int

	KeepAliveTimeout, RequestTimeout time.Duration
}

func (c *ConnectionConfig) prepare() {
	if c.SenderBufferSize <= 0 {
		c.SenderBufferSize = defaultSenderBufferSize
	}
	if c.KeepAliveTimeout <= 0 {
		c.KeepAliveTimeout = defaultKeepAliveTimeout
	}
	if c.RequestTimeout <= 0 {
		c.RequestTimeout = defaultRequestTimeout
	}
}

func (c *ConnectionConfig) getNetConn() (net.Conn, error) {
	if c.Conn != nil {
		return c.Conn, nil
	}
	return net.Dial("tcp", c.Host+":"+strconv.FormatUint(c.Port, 10))
}
