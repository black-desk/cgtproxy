//go:build debug
// +build debug

package table

import (
	"go.uber.org/zap"

	. "github.com/black-desk/deepin-network-proxy-manager/internal/core/table/internal"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/google/nftables/expr"
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
