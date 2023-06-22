package nftman

import (
	"errors"
	"github.com/black-desk/cgtproxy/internal/consts"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
	"net"
	"syscall"
)

func (t *Table) ignoreNoBufferSpaceAvailable(perr *error) {
	var errno syscall.Errno
	if errors.As(*perr, &errno) && errno == syscall.ENOBUFS {
		*perr = nil
		t.log.Errorw("ENOBUFS occurred.")
		//FIXME: https://github.com/google/nftables/issues/103
	}
}

func (t *Table) initStructure() (err error) {
	defer Wrap(&err, "Failed to flush initial content of nft table.")

	t.log.Debug("Initialing nft table structure.")

	var conn *nftables.Conn
	conn, err = nftables.New()
	if err != nil {
		return
	}

	t.table = conn.AddTable(&nftables.Table{
		Name:   consts.NftTableName,
		Family: nftables.TableFamilyINet,
	})

	err = t.initIPV4BypassSet(conn)
	if err != nil {
		return
	}

	err = t.initIPV6BypassSet(conn)
	if err != nil {
		return
	}

	t.initProtoSet()

	err = t.initCgroupMap(conn)
	if err != nil {
		return
	}

	err = t.initMarkMap(conn)
	if err != nil {
		return
	}

	err = t.initMarkDNSMap(conn)
	if err != nil {
		return
	}

	t.policy = nftables.ChainPolicyAccept

	err = t.initOutputMangleChain(conn)
	if err != nil {
		return
	}

	err = t.initOutputNATChain(conn)
	if err != nil {
		return
	}

	err = t.initPreroutingChain(conn)
	if err != nil {
		return
	}

	err = conn.Flush()
	t.ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	t.log.Debug("nft table structure initialized.")

	t.DumpNFTableRules()

	return
}

func (t *Table) initIPV4BypassSet(conn *nftables.Conn) (err error) {
	t.ipv4BypassSet = &nftables.Set{
		Table:        t.table,
		Name:         "bypass",
		KeyType:      nftables.TypeIPAddr,
		KeyByteOrder: binaryutil.BigEndian,
		Interval:     true,
	}

	elements := []nftables.SetElement{{
		Key:         net.ParseIP("0.0.0.0").To4(),
		IntervalEnd: true,
	}}

	for i := range t.bypassIPv4 {
		bypass := t.bypassIPv4[i]
		ip := net.ParseIP(bypass)

		if ip != nil {
			elements = append(elements,
				nftables.SetElement{
					Key: ip.To4(),
				},
				nftables.SetElement{
					Key:         t.nextIP(ip).To4(),
					IntervalEnd: true,
				},
			)
			continue
		}

		_, cidr, err := net.ParseCIDR(bypass)
		if err != nil {
			// This should never happened,
			// as string has been checked by validator.
			panic(err)
		}

		elements = append(elements,
			nftables.SetElement{
				Key: cidr.IP.To4(),
			},
			nftables.SetElement{
				Key:         t.nextIP(t.lastIP(cidr).To4()),
				IntervalEnd: true,
			},
		)
	}

	err = conn.AddSet(t.ipv4BypassSet, elements)
	if err != nil {
		return
	}

	return
}

func (t *Table) nextIP(ip net.IP) (ret net.IP) {
	next := make(net.IP, len(ip))
	copy(next, ip)

	for i := range next {
		i = len(next) - i - 1
		old := next[i]
		next[i] += 1
		if next[i] >= old {
			break
		}
	}

	t.log.Debugf("Next IP of %s is %s", ip.String(), next.String())

	ret = next
	return
}

func (t *Table) lastIP(ipnet *net.IPNet) (ret net.IP) {
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)

	for i := range ip {
		ip[i] |= ^ipnet.Mask[i]
	}

	t.log.Debugf("Last IP in net %s is %s", ipnet.String(), ip.String())

	ret = ip
	return
}

func (t *Table) initIPV6BypassSet(conn *nftables.Conn) (err error) {
	t.ipv6BypassSet = &nftables.Set{
		Table:        t.table,
		Name:         "bypass6",
		KeyType:      nftables.TypeIP6Addr,
		KeyByteOrder: binaryutil.BigEndian,
		Interval:     true,
	}

	elements := []nftables.SetElement{{
		Key:         net.ParseIP("::").To16(),
		IntervalEnd: true,
	}}

	for i := range t.bypassIPv6 {
		bypass := t.bypassIPv6[i]
		ip := net.ParseIP(bypass)
		if ip != nil {
			elements = append(elements,
				nftables.SetElement{
					Key: ip.To16(),
				},
				nftables.SetElement{
					Key:         t.nextIP(ip.To16()),
					IntervalEnd: true,
				},
			)
			continue
		}

		_, cidr, err := net.ParseCIDR(bypass)
		if err != nil {
			// This should never happened,
			// as string has been checked by validator.
			panic(err)
		}

		elements = append(elements,
			nftables.SetElement{
				Key: cidr.IP.To16(),
			},
			nftables.SetElement{
				Key:         t.nextIP(t.lastIP(cidr).To16()),
				IntervalEnd: true,
			},
		)
	}

	err = conn.AddSet(t.ipv6BypassSet, elements)
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

func (t *Table) initCgroupMap(conn *nftables.Conn) (err error) {
	t.cgroupMap = &nftables.Set{
		Table:        t.table,
		Name:         "cgroup-vmap",
		KeyType:      nftables.TypeCGroupV2,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	t.cgroupMapElement = make(map[string]nftables.SetElement)

	err = conn.AddSet(t.cgroupMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

func (t *Table) initMarkMap(conn *nftables.Conn) (err error) {
	t.markTproxyMap = &nftables.Set{
		Table:        t.table,
		Name:         "mark-vmap",
		KeyType:      nftables.TypeMark,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	err = conn.AddSet(t.markTproxyMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

func (t *Table) initMarkDNSMap(conn *nftables.Conn) (err error) {
	t.markDNSMap = &nftables.Set{
		Table:        t.table,
		Name:         "mark-dns-vmap",
		KeyType:      nftables.TypeMark,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	err = conn.AddSet(t.markDNSMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

func (t *Table) initOutputMangleChain(conn *nftables.Conn) (err error) {
	// type filter hook prerouting priority mangle; policy accept;
	t.outputMangleChain = conn.AddChain(&nftables.Chain{
		Table:    t.table,
		Name:     "output-mangle",
		Type:     nftables.ChainTypeRoute,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &t.policy,
	})

	err = t.fillOutputMangleChain(conn, t.outputMangleChain)
	if err != nil {
		return
	}

	return
}

func (t *Table) fillOutputMangleChain(
	conn *nftables.Conn, chain *nftables.Chain,
) (
	err error,
) {
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
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	})

	// ip daddr @bypass return
	exprs = []expr.Any{
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

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	})

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

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	})

	// meta l4proto != { tcp, udp } return

	err = conn.AddSet(t.protoSet, t.protoSetElement)
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

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	})

	return
}

func (t *Table) initOutputNATChain(conn *nftables.Conn) (err error) {
	// type nat hook prerouting priority -100; policy accept;
	t.outputNATChain = conn.AddChain(&nftables.Chain{
		Table:    t.table,
		Name:     "output-nat",
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityNATDest,
		Policy:   &t.policy,
	})

	// meta mark vmap @
	exprs := []expr.Any{
		&expr.Meta{
			Key:      expr.MetaKeyMARK,
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set mark-vmap dreg 0
			SourceRegister: 1,
			IsDestRegSet:   true,
			SetName:        t.markDNSMap.Name,
			SetID:          t.markDNSMap.ID,
		},
	}
	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.outputNATChain,
		Exprs: exprs,
	})

	return
}

func (t *Table) initPreroutingChain(conn *nftables.Conn) (err error) {
	// type route hook output priority mangle; policy accept;
	t.preroutingChain = conn.AddChain(&nftables.Chain{
		Table:    t.table,
		Name:     "prerouting",
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &t.policy,
	})

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

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.preroutingChain,
		Exprs: exprs,
	})

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

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.preroutingChain,
		Exprs: exprs,
	})

	// meta mark vmap @mark-vmap
	exprs = []expr.Any{
		&expr.Meta{
			Key:      expr.MetaKeyMARK,
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set mark-vmap dreg 0
			SourceRegister: 1,
			IsDestRegSet:   true,
			SetName:        t.markTproxyMap.Name,
			SetID:          t.markTproxyMap.ID,
		},
	}
	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: t.preroutingChain,
		Exprs: exprs,
	})

	return
}
