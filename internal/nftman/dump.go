//go:build !debug
// +build !debug

package nftman

import (
	"github.com/google/nftables/expr"
)

func (t *Table) DumpNFTableRules() {
	return
}

func addDebugCounter(exprs []expr.Any) []expr.Any {
	return exprs
}
