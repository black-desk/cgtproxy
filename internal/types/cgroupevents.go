// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package types

type CGroupEvents struct {
	// Initial marks the complete first scan, even when Events is empty.
	// The monitor sends it before any incremental events.
	Initial bool
	Events  []CGroupEvent
	Result  chan<- error
}
