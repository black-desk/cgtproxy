package table

import (
	"errors"
	"math/rand"

	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

func (t *Table) initChecks() (err error) {
	if t.conn == nil {
		err = errors.New("`Table` need a nftables conn.")
		return
	}

	return
}

func (t *Table) initStructure() {
	t.table = &nftables.Table{
		Name:   "deepin-network-proxy",
		Family: nftables.TableFamilyINet,
	}

	t.ipv4BypassSet = &nftables.Set{
		Table:    t.table,
		Constant: true,
		Name:     "bypass",
		KeyType:  nftables.TypeIPAddr,
	}

	t.ipv6BypassSet = &nftables.Set{
		Table:    t.table,
		Constant: true,
		Name:     "bypass6",
		KeyType:  nftables.TypeIP6Addr,
	}

	t.bypassCgroupSets = map[uint32]cgroupSet{}

	t.protoSet = &nftables.Set{
		Table:     t.table,
		Anonymous: true,
		ID:        rand.Uint32(),
		KeyType:   nftables.TypeInetProto,
	}
	t.protoSetElement = []nftables.SetElement{
		{Key: []byte{unix.IPPROTO_TCP}},
		{Key: []byte{unix.IPPROTO_UDP}},
	}

	t.cgroupMaps = map[uint32]cgroupSet{}

	t.tproxyChains = []*nftables.Chain{}

	t.tproxyRules = map[string][]*nftables.Rule{}

	t.policy = nftables.ChainPolicyAccept

	// type filter hook prerouting priority mangle; policy accept;
	t.outputChain = &nftables.Chain{
		Table:    t.table,
		Name:     "output",
		Type:     nftables.ChainTypeRoute,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &t.policy,
	}
	t.outputRules = []*nftables.Rule{
		{ // ip daddr @bypass return
			Table: t.table,
			Chain: t.outputChain,
			Exprs: []expr.Any{
				&expr.Meta{ // meta load nfproto => reg 1
					Key:      expr.MetaKeyNFPROTO,
					Register: 1,
				},
				&expr.Cmp{ // cmp eq reg 1 0x00000002
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     []byte{0x00000002},
				},
				&expr.Payload{ // payload load 4b @ network header + 16 => reg 1
					OperationType: expr.PayloadLoad,
					DestRegister:  1,
					Base:          expr.PayloadBaseNetworkHeader,
					Offset:        16,
					Len:           4,
				},
				&expr.Lookup{ // lookup reg 1 set bypass
					SourceRegister: 1,
					SetName:        t.ipv4BypassSet.Name,
				},
				&expr.Verdict{ // immediate reg 0 return
					Kind: expr.VerdictReturn,
				},
			},
		},
		{ // ip6 daddr @bypass6 return
			Table: t.table,
			Chain: t.outputChain,
			Exprs: []expr.Any{
				&expr.Meta{ // meta load nfproto => reg 1
					Key:      expr.MetaKeyNFPROTO,
					Register: 1,
				},
				&expr.Cmp{ // cmp eq reg 1 0x0000000a
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     []byte{0x0000000a},
				},
				&expr.Payload{ // payload load 4b @ network header + 16 => reg 1
					OperationType: expr.PayloadLoad,
					DestRegister:  1,
					Base:          expr.PayloadBaseNetworkHeader,
					Offset:        24,
					Len:           16,
				},
				&expr.Lookup{ // lookup reg 1 set bypass
					SourceRegister: 1,
					SetName:        t.ipv4BypassSet.Name,
				},
				&expr.Verdict{ // immediate reg 0 return
					Kind: expr.VerdictReturn,
				},
			},
		},
		{ // meta l4proto != { tcp, udp } return # handle 1000
			Table:  t.table,
			Chain:  t.outputChain,
			Handle: consts.RuleInsertHandle,
			Exprs: []expr.Any{
				&expr.Meta{ // meta load l4proto => reg 1
					Key:      expr.MetaKeyL4PROTO,
					Register: 1,
				},
				&expr.Lookup{ // lookup reg 1 set __set%d
					SourceRegister: 1,
					SetID:          t.protoSet.ID,
					Invert:         true,
				},
				&expr.Verdict{ // immediate reg 0 return
					Kind: expr.VerdictReturn,
				},
			},
		},
		{ // meta mark set ...
			Table: t.table,
			Chain: t.outputChain,
			Exprs: []expr.Any{
				&expr.Immediate{ // immediate reg 1 ...
					Register: 1,
					Data:     binaryutil.NativeEndian.PutUint32(t.rerouteMark),
				},
			},
		},
	}

	// type route hook output priority mangle; policy accept;
	t.preroutingChain = &nftables.Chain{
		Table:    t.table,
		Name:     "prerouting",
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &t.policy,
	}
	t.preroutingRules = []*nftables.Rule{
		{ // ip daddr @bypass return
			Table: t.table,
			Chain: t.outputChain,
			Exprs: []expr.Any{
				&expr.Meta{ // meta load nfproto => reg 1
					Key:      expr.MetaKeyNFPROTO,
					Register: 1,
				},
				&expr.Cmp{ // cmp eq reg 1 0x00000002
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     []byte{0x00000002},
				},
				&expr.Payload{ // payload load 4b @ network header + 16 => reg 1
					OperationType: expr.PayloadLoad,
					DestRegister:  1,
					Base:          expr.PayloadBaseNetworkHeader,
					Offset:        16,
					Len:           4,
				},
				&expr.Lookup{ // lookup reg 1 set bypass
					SourceRegister: 1,
					SetName:        t.ipv4BypassSet.Name,
				},
				&expr.Verdict{ // immediate reg 0 return
					Kind: expr.VerdictReturn,
				},
			},
		},
		{ // ip6 daddr @bypass6 return
			Table: t.table,
			Chain: t.outputChain,
			Exprs: []expr.Any{
				&expr.Meta{ // meta load nfproto => reg 1
					Key:      expr.MetaKeyNFPROTO,
					Register: 1,
				},
				&expr.Cmp{ // cmp eq reg 1 0x0000000a
					Op:       expr.CmpOpEq,
					Register: 1,
					Data:     []byte{0x0000000a},
				},
				&expr.Payload{ // payload load 4b @ network header + 16 => reg 1
					OperationType: expr.PayloadLoad,
					DestRegister:  1,
					Base:          expr.PayloadBaseNetworkHeader,
					Offset:        24,
					Len:           16,
				},
				&expr.Lookup{ // lookup reg 1 set bypass
					SourceRegister: 1,
					SetName:        t.ipv4BypassSet.Name,
				},
				&expr.Verdict{ // immediate reg 0 return
					Kind: expr.VerdictReturn,
				},
			},
		},
		{ // accept # handle 1000
			Table:  t.table,
			Chain:  t.outputChain,
			Handle: consts.RuleInsertHandle,
			Exprs: []expr.Any{
				&expr.Verdict{ // accept
					Kind: expr.VerdictAccept,
				},
			},
		},
	}
}
