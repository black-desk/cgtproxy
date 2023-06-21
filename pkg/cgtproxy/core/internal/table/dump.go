//go:build !debug
// +build !debug

package table

import (
	"github.com/google/nftables/expr"
)

func DumpNFTableRules() {
	return
}

func addDebugCounter(exprs []expr.Any) []expr.Any {
	return exprs
}
