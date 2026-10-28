// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package connector

import (
	. "github.com/black-desk/cgtproxy/internal/errwrap"
	"github.com/google/nftables"
)

type Connector struct {
	conn *nftables.Conn
}

type Opt = (func(*Connector) (*Connector, error))

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/internal/nftman/connector.Connector -as interfaces.NetlinkConnector -o ../../interfaces/netlinkconnector.go

func New(...Opt) (ret *Connector, err error) {
	return &Connector{}, nil
}

func (c *Connector) Connect() (ret *nftables.Conn, err error) {
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

func (c *Connector) Release() (err error) {
	defer Wrap(&err, "release lasting netlink connection")

	if c.conn == nil {
		return nil
	}

	err = c.conn.CloseLasting()
	c.conn = nil
	return
}
