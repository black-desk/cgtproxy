// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package types

type CGroupEvents struct {
	Events []CGroupEvent
	Result chan<- error
}
