// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package connector

import (
	"github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/google/nftables"
)

type Connector struct {
	conn *nftables.Conn
}

func (c *Connector) Connect() (ret *nftables.Conn, err error) {
	defer errwrap.Wrap(&err, "new lasting netlink connection")

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

func (c *Connector) Release() (err error) {
	defer errwrap.Wrap(&err, "release lasting netlink connection")

	if c.conn == nil {
		return nil
	}

	err = c.conn.CloseLasting()
	c.conn = nil
	return
}
