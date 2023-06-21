//go:build debug
// +build debug

package table

import (
	. "github.com/black-desk/cgtproxy/internal/log"
	. "github.com/black-desk/cgtproxy/pkg/cgtproxy/core/internal/table/internal"
	"github.com/google/nftables/expr"
	"go.uber.org/zap"
)

func DumpNFTableRules() {
	Log.WithOptions(zap.AddStacktrace(zap.DebugLevel)).Debugw(
		"Dump nft ruleset.",
		"content", GetNFTableRules(),
	)

	return
}

func addDebugCounter(exprs []expr.Any) []expr.Any {
	return append([]expr.Any{&expr.Counter{}}, exprs...)
}
