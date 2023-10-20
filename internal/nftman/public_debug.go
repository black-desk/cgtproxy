//go:build debug
// +build debug

package nftman

import (
	"go.uber.org/zap"
)

func (nft *NFTMan) DumpNFTableRules() {
	nft.log.WithOptions(zap.AddStacktrace(zap.DebugLevel)).Debugw(
		"Dump nft ruleset.",
		"content", getNFTableRules(),
	)

	return
}
