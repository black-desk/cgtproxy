// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import "errors"

var (
	ErrNftableConnMissing = errors.New("`nftables.Conn` is missing.")
	ErrRerouteMarkMissing = errors.New("reroute mark is missing.")
	ErrLoggerMissing      = errors.New("logger is missing.")
	ErrCGroupRootMissing  = errors.New("cgroupv2 file system mount point is missing.")
	ErrConnFactoryMissing = errors.New("netlink conn factory is missing.")
)
