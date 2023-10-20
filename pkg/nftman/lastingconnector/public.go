package lastingconnector

import (
	"github.com/google/nftables"
)

func (c *LastingConnector) Connect() (ret *nftables.Conn, err error) {
	if c.conn != nil {
		ret = c.conn
		return
	}

	var conn *nftables.Conn
	conn, err = nftables.New(nftables.AsLasting())
	if err != nil {
		return
	}

	c.conn = conn
	ret = c.conn
	return
}

func (c *LastingConnector) Release() (err error) {
	err = c.conn.CloseLasting()
	c.conn = nil
	return
}
