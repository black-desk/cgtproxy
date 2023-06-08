package table

import (
	"errors"
	"net"
	"syscall"

	"github.com/black-desk/cgtproxy/internal/consts"
	. "github.com/black-desk/cgtproxy/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

func ignoreNoBufferSpaceAvailable(perr *error) {
	var errno syscall.Errno
	if errors.As(*perr, &errno) && errno == syscall.ENOBUFS {
		*perr = nil
		Log.Errorw("ENOBUFS occurred.")
		//FIXME: https://github.com/google/nftables/issues/103
	}
}

func (t *Table) initChecks() (err error) {
	defer Wrap(&err)

	if t.conn == nil {
		err = ErrMissingNftableConn
		return
	}

	return
}

func (t *Table) initStructure() (err error) {
	defer Wrap(&err, "Failed to flush initial content of nft table.")

	Log.Debug("Initialing nft table structure.")

	t.table = t.conn.AddTable(&nftables.Table{
		Name:   consts.NftTableName,
		Family: nftables.TableFamilyINet,
	})

	err = t.initIPV4BypassSet()
	if err != nil {
		return
	}

	err = t.initIPV6BypassSet()
	if err != nil {
		return
	}

	t.initProtoSet()

	err = t.initCgroupMap()
	if err != nil {
		return
	}

	err = t.initMarkMap()
	if err != nil {
		return
	}

	t.policy = nftables.ChainPolicyAccept

	err = t.initOutputChain()
	if err != nil {
		return
	}

	err = t.initPreroutingChain()
	if err != nil {
		return
	}

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	Log.Debug("nft table structure initialized.")

	DumpNFTableRules()

	return
}

func (t *Table) initIPV4BypassSet() (err error) {
	t.ipv4BypassSet = &nftables.Set{
		Table:        t.table,
		Name:         "bypass",
		KeyType:      nftables.TypeIPAddr,
		KeyByteOrder: binaryutil.BigEndian,
	}

	elements := []nftables.SetElement{}

	for i := range t.bypassIPv4 {
		elements = append(elements, nftables.SetElement{
			Key: []byte(net.ParseIP(t.bypassIPv4[i]).To4()),
		})
	}

	err = t.conn.AddSet(t.ipv4BypassSet, elements)
	if err != nil {
		return
	}

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}

func (t *Table) initIPV6BypassSet() (err error) {
	t.ipv6BypassSet = &nftables.Set{
		Table:        t.table,
		Name:         "bypass6",
		KeyType:      nftables.TypeIP6Addr,
		KeyByteOrder: binaryutil.BigEndian,
	}

	elements := []nftables.SetElement{}

	for i := range t.bypassIPv6 {
		elements = append(elements, nftables.SetElement{
			Key: []byte(net.ParseIP(t.bypassIPv6[i]).To16()),
		})
	}

	err = t.conn.AddSet(t.ipv6BypassSet, elements)
	if err != nil {
		return
	}

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}

func (t *Table) initProtoSet() {
	t.protoSet = &nftables.Set{
		Table:     t.table,
		Anonymous: true,
		Constant:  true,
		KeyType:   nftables.TypeInetProto,
	}

	t.protoSetElement = []nftables.SetElement{
		{Key: []byte{unix.IPPROTO_TCP}},
		{Key: []byte{unix.IPPROTO_UDP}},
	}
}

func (t *Table) initCgroupMap() (err error) {
	t.cgroupMap = &nftables.Set{
		Table:        t.table,
		Name:         "cgroup-vmap",
		KeyType:      nftables.TypeCGroupV2,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	t.cgroupMapElement = make(map[string]nftables.SetElement)

	err = t.conn.AddSet(t.cgroupMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}

func (t *Table) initMarkMap() (err error) {
	t.markMap = &nftables.Set{
		Table:        t.table,
		Name:         "mark-vmap",
		KeyType:      nftables.TypeMark,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	err = t.conn.AddSet(t.markMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}

func (t *Table) initOutputChain() (err error) {
	// type filter hook prerouting priority mangle; policy accept;
	t.outputChain = t.conn.AddChain(&nftables.Chain{
		Table:    t.table,
		Name:     "output",
		Type:     nftables.ChainTypeRoute,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &t.policy,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	// ip daddr @bypass return

	exprs := []expr.Any{
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
			SetID:          t.ipv4BypassSet.ID,
			SetName:        t.ipv4BypassSet.Name,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	t.conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.outputChain,
		Exprs: exprs,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	// ip6 daddr @bypass6 return

	exprs = []expr.Any{
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
			SetID:          t.ipv6BypassSet.ID,
			SetName:        t.ipv6BypassSet.Name,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	t.conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.outputChain,
		Exprs: exprs,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	// meta l4proto != { tcp, udp } return

	err = t.conn.AddSet(t.protoSet, t.protoSetElement)
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
			SetID:          t.protoSet.ID,
			SetName:        t.protoSet.Name,
			Invert:         true,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	t.conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.outputChain,
		Exprs: exprs,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}

func (t *Table) initPreroutingChain() (err error) {
	// type route hook output priority mangle; policy accept;
	t.preroutingChain = t.conn.AddChain(&nftables.Chain{
		Table:    t.table,
		Name:     "prerouting",
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &t.policy,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	// ip daddr @bypass return
	exprs := []expr.Any{
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
			SetID:          t.ipv4BypassSet.ID,
			SetName:        t.ipv4BypassSet.Name,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	t.conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.preroutingChain,
		Exprs: exprs,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	// ip6 daddr @bypass6 return
	exprs = []expr.Any{
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
			SetID:          t.ipv6BypassSet.ID,
			SetName:        t.ipv6BypassSet.Name,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	t.conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.preroutingChain,
		Exprs: exprs,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	// meta mark vmap @mark-vmap
	exprs = []expr.Any{
		&expr.Meta{
			Key:      expr.MetaKeyMARK,
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set cgroup-map-x dreg 0
			SourceRegister: 1,
			IsDestRegSet:   true,
			SetName:        t.markMap.Name,
			SetID:          t.markMap.ID,
		},
	}
	exprs = addDebugCounter(exprs)

	t.conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.preroutingChain,
		Exprs: exprs,
	})

	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}
