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
