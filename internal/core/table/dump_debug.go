//go:build debug
// +build debug

package table

import (
	. "github.com/black-desk/deepin-network-proxy-manager/internal/core/table/internal"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
)

func DumpNFTableRules() {
	Log.Debugw("Dump nft ruleset.",
		"content", GetNFTableRules(),
	)

	return
}
