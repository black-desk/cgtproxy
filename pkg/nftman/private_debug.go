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
