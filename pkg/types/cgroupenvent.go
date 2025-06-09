// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package types

type CgroupEventType uint8

const (
	CgroupEventTypeNew    CgroupEventType = iota // New
	CgroupEventTypeDelete                        // Delete
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=CgroupEventType -linecomment

type CGroupEvent struct {
	Path      string
	EventType CgroupEventType
}
