// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !debug
// +build !debug

package nftman

import (
	"github.com/google/nftables/expr"
)

func addDebugCounter(exprs []expr.Any) []expr.Any {
	return exprs
}

func (nft *NFTManager) dumpNFTableRules() {
	return
}
