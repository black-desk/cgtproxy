// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build debug
// +build debug

package nftman

import (
	"github.com/google/nftables/expr"
	"go.uber.org/zap"
)

func addDebugCounter(exprs []expr.Any) []expr.Any {
	return append([]expr.Any{&expr.Counter{}}, exprs...)
}

func (nft *NFTManager) dumpNFTableRules() {
	nft.log.WithOptions(zap.AddStacktrace(zap.DebugLevel)).Debugw(
		"Dump nft ruleset.",
		"content", getNFTableRules(),
	)

	return
}
