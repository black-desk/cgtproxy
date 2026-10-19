// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package lastingconnector

import "github.com/google/nftables"

type LastingConnector struct {
	conn *nftables.Conn
}

type Opt = (func(*LastingConnector) (*LastingConnector, error))

func New(...Opt) (ret *LastingConnector, err error) {
	return &LastingConnector{}, nil
}
