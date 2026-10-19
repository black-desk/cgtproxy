// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

const (
	DefaultConfig = `
version: 1
cgroup-root: AUTO
route-table: 300
rules:
  - match: \/.*
    direct: true
`
	IPv4LocalhostStr = "127.0.0.1"
)
