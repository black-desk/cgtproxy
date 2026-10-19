// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package lastingconnector

import (
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
)

func (c *LastingConnector) Connect() (ret *nftables.Conn, err error) {
	defer Wrap(&err, "new lasting netlink connection")

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
	defer Wrap(&err, "release lasting netlink connection")

	err = c.conn.CloseLasting()
	c.conn = nil
	return
}
