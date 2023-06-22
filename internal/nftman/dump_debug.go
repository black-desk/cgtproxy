//go:build debug
// +build debug

package nftman

import (
	. "github.com/black-desk/cgtproxy/internal/nftman/internal"
	"github.com/google/nftables/expr"
	"go.uber.org/zap"
)

func (t *Table) DumpNFTableRules() {
	t.log.WithOptions(zap.AddStacktrace(zap.DebugLevel)).Debugw(
		"Dump nft ruleset.",
		"content", GetNFTableRules(),
	)

	return
}

func addDebugCounter(exprs []expr.Any) []expr.Any {
	return append([]expr.Any{&expr.Counter{}}, exprs...)
}
