// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build debug

package nftman

import "github.com/google/nftables/expr"

// Debug builds prepend "counter" to each rule, before its match expressions.
// nft: counter <matches> <statements>
// The counter therefore counts packets reaching the rule, even on a mismatch.
func addDebugCounter(exprs []expr.Any) []expr.Any {
	return append([]expr.Any{&expr.Counter{}}, exprs...)
}
