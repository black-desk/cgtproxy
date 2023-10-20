//go:build debug
// +build debug

package nftman

import (
	"go.uber.org/zap"
)

func (t *Table) DumpNFTableRules() {
	t.log.WithOptions(zap.AddStacktrace(zap.DebugLevel)).Debugw(
		"Dump nft ruleset.",
		"content", getNFTableRules(),
	)

	return
}
