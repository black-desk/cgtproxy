// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

// addCommonFilters adds the filtering prefix shared by local OUTPUT and
// forwarded PREROUTING traffic, before either selects a proxy target.
func (nft *NFTManager) addCommonFilters(
	conn *nftables.Conn, chain *nftables.Chain,
) (
	err error,
) {
	nft.log.Debug("Refilling output chain.")

	// ct direction == reply return
	exprs := []expr.Any{
		&expr.Ct{ // ct load direction => reg 1
			Register: 1,
			Key:      expr.CtKeyDIRECTION,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00000001
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{0x00000001}, // IP_CT_DIR_REPLY
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}
	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: chain,
		Exprs: exprs,
	})

	nft.addBypassRules(conn, chain, false)

	// meta l4proto != { tcp, udp } return

	nft.protoSet.ID = 0
	err = conn.AddSet(nft.protoSet, nft.protoSetElement)
	if err != nil {
		return
	}

	exprs = []expr.Any{
		&expr.Meta{ // meta load l4proto => reg 1
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set __set%d
			SourceRegister: 1,
			SetID:          nft.protoSet.ID,
			SetName:        nft.protoSet.Name,
			Invert:         true,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: chain,
		Exprs: exprs,
	})

	return
}

func (nft *NFTManager) initProtoSet() {
	nft.protoSet = &nftables.Set{
		Table:     nft.table,
		Anonymous: true,
		Constant:  true,
		KeyType:   nftables.TypeInetProto,
	}

	nft.protoSetElement = []nftables.SetElement{
		{Key: []byte{unix.IPPROTO_TCP}},
		{Key: []byte{unix.IPPROTO_UDP}},
	}
}

func (nft *NFTManager) addReturnRule(conn *nftables.Conn, chain *nftables.Chain, filters ...expr.Any) {
	filters = append(filters, &expr.Verdict{Kind: expr.VerdictReturn})
	conn.AddRule(&nftables.Rule{Table: nft.table, Chain: chain, Exprs: addDebugCounter(filters)})
}
