package conn

import (
	"fmt"
	"net"
	"net/url"

	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	"go.uber.org/zap"
)

type JournalConn struct {
	address  string
	unixAddr *net.UnixAddr
	limit    int
	unixConn *net.UnixConn
}

type Opt = (func(*JournalConn) (*JournalConn, error))

func New(opts ...Opt) (ret *JournalConn, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Failed to create new connection to journald:\n%w",
			err,
		)
	}()

	conn := &JournalConn{
		address: "/run/systemd/journal/socket",
		limit:   -1,
	}
	for i := range opts {
		conn, err = opts[i](conn)
		if err != nil {
			conn = nil
			return
		}
	}

	conn.unixAddr, err = net.ResolveUnixAddr("unixgram", conn.address)
	if err != nil {
		err = fmt.Errorf(location.Capture(),
			"Failed to resolve address of unix domain socket:\n%w",
			err,
		)
		return
	}

	err = conn.connect()
	if err != nil {
		return
	}

	ret = conn

	return
}

func (c *JournalConn) connect() (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Failed to connect to journald:\n%w",
			err,
		)
	}()

	var addr *net.UnixAddr
	addr, err = net.ResolveUnixAddr("unixgram", "")
	if err != nil {
		err = fmt.Errorf(location.Capture(),
			"Failed to resolve address of local unix domain socket:\n%w",
			err,
		)
		return
	}

	c.unixConn, err = net.ListenUnixgram("unixgram", addr)

	return
}

func WithAddress(address string) Opt {
	return func(conn *JournalConn) (ret *JournalConn, err error) {
		conn.address = address
		ret = conn
		return
	}
}

func WithLimit(limit int) Opt {
	return func(conn *JournalConn) (ret *JournalConn, err error) {
		conn.limit = limit
		ret = conn
		return
	}
}

func init() {
	err := zap.RegisterSink("journal", func(url *url.URL) (zap.Sink, error) {
		return New(WithAddress(url.Path))
	})
	if err != nil {
		panic("zap-journal: Failed to register sink: " + err.Error())
	}
}
