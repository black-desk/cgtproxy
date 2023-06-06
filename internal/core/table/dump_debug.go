//go:build debug
// +build debug

package table

import (
	"go.uber.org/zap"

	. "github.com/black-desk/deepin-network-proxy-manager/internal/core/table/internal"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
)

func DumpNFTableRules() {
	Log.WithOptions(zap.AddStacktrace(zap.DebugLevel)).Debugw(
		"Dump nft ruleset.",
		"content", GetNFTableRules(),
	)

	return
}
