package amigo

import (
	"net"
	"net/url"
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

func ConfigFromURL(URL string) (*ConnectionConfig, error) {
	tmp, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	host := tmp.Hostname()
	port, _ := strconv.ParseUint(tmp.Port(), 10, 64)

	var user, pass string
	if tmp.User != nil {
		user = tmp.User.Username()
		pass, _ = tmp.User.Password()
	}

	senderBufferSize, _ := strconv.Atoi(tmp.Query().Get("SenderBufferSize"))

	keepAliveTimeout, _ := strconv.ParseInt(tmp.Query().Get("KeepAliveTimeout"), 10, 64)
	requestTimeout, _ := strconv.ParseInt(tmp.Query().Get("RequestTimeout"), 10, 64)

	return &ConnectionConfig{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,

		SenderBufferSize: senderBufferSize,
		KeepAliveTimeout: time.Duration(keepAliveTimeout),
		RequestTimeout:   time.Duration(requestTimeout),
	}, nil
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
