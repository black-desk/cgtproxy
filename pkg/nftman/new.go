package nftman

import (
	"net"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/cgtproxy/pkg/nftman/connector"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

type NFTManager struct {
	cgroupRoot config.CGroupRoot
	bypassIPv4 []string
	bypassIPv6 []string
	log        *zap.SugaredLogger

	connector interfaces.NetlinkConnector

	table *nftables.Table

	ipv4BypassSet *nftables.Set
	ipv6BypassSet *nftables.Set

	// NOTE(black_desk):
	// When use AddSet to add anonymous protoSet into nftable,
	// we should reset protoSet.ID to 0
	// to let nftables reallocate ID for this anonymous set.
	protoSet        *nftables.Set
	protoSetElement []nftables.SetElement // keep anonymous set elements

	policy nftables.ChainPolicy

	cgroupMap        *nftables.Set
	cgroupMapElement map[string]nftables.SetElement

	markTproxyMap *nftables.Set
	markDNSMap    *nftables.Set

	outputMangleChain *nftables.Chain
	outputNATChain    *nftables.Chain
	preroutingChain   *nftables.Chain
}

type Opt = (func(*NFTManager) (*NFTManager, error))

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/pkg/nftman.NFTManager -as interfaces.NFTManager -o ../interfaces/nftman.go
func New(opts ...Opt) (ret *NFTManager, err error) {
	defer Wrap(&err, "create nft table mananger")

	t := &NFTManager{}

	for i := range opts {
		t, err = opts[i](t)
		if err != nil {
			t = nil
			return
		}
	}

	if t.connector == nil {
		var ctr *connector.Connector
		ctr, err = connector.New()
		if err != nil {
			return
		}

		t.connector = ctr
	}

	if t.log == nil {
		t.log = zap.NewNop().Sugar()
	}

	ret = t

	t.log.Debugw("NFTManager created.")
	return
}

func WithBypass(bypass config.Bypass) Opt {
	return func(table *NFTManager) (ret *NFTManager, err error) {
		if bypass == nil {
			ret = table
			return
		}

		for i := range bypass {
			ip := net.ParseIP(bypass[i])

			if ip == nil {
				ip, _, err = net.ParseCIDR(bypass[i])
				if err != nil {
					return
				}
			}

			if ip.To4() != nil {
				table.bypassIPv4 = append(table.bypassIPv4, bypass[i])
			} else if ip.To16() != nil {
				table.bypassIPv6 = append(table.bypassIPv6, bypass[i])
			} else {
				panic("this should never happened, check validator.")
			}
		}

		return table, nil
	}
}

func WithCgroupRoot(root config.CGroupRoot) Opt {
	return func(table *NFTManager) (ret *NFTManager, err error) {
		if root == "" {
			err = ErrCGroupRootMissing
			return
		}

		table.cgroupRoot = root
		return table, nil
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(nft *NFTManager) (ret *NFTManager, err error) {
		if log == nil {
			err = ErrLoggerMissing
			return
		}

		nft.log = log
		return nft, nil
	}
}

func WithConnFactory(f interfaces.NetlinkConnector) Opt {
	return func(nft *NFTManager) (ret *NFTManager, err error) {
		if f == nil {
			err = ErrConnFactoryMissing
			return
		}

		nft.connector = f
		ret = nft
		return
	}
}

func (nft *NFTManager) InitStructure() (err error) {
	defer Wrap(&err, "flush initial content")

	nft.log.Debug("Initialing nft table structure.")

	var conn *nftables.Conn
	conn, err = nft.connector.Connect()
	if err != nil {
		return
	}

	nft.table = conn.CreateTable(&nftables.Table{
		Name:   NftTableName,
		Family: nftables.TableFamilyINet,
	})

	err = conn.Flush()
	if err != nil {
		Wrap(&err, "create table")
		return
	}

	err = nft.initIPV4BypassSet(conn)
	if err != nil {
		return
	}

	err = nft.initIPV6BypassSet(conn)
	if err != nil {
		return
	}

	nft.initProtoSet()

	err = nft.initCgroupMap(conn)
	if err != nil {
		return
	}

	err = nft.initMarkMap(conn)
	if err != nil {
		return
	}

	err = nft.initMarkDNSMap(conn)
	if err != nil {
		return
	}

	nft.policy = nftables.ChainPolicyAccept

	err = nft.initOutputMangleChain(conn)
	if err != nil {
		return
	}

	err = nft.initOutputNATChain(conn)
	if err != nil {
		return
	}

	err = nft.initPreroutingChain(conn)
	if err != nil {
		return
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	nft.log.Debug("nft table structure initialized.")

	nft.dumpNFTableRules()

	return
}

func (nft *NFTManager) initIPV4BypassSet(conn *nftables.Conn) (err error) {
	defer Wrap(&err, "prepare ipv4 bypass set")

	nft.ipv4BypassSet = &nftables.Set{
		Table:        nft.table,
		Name:         "bypass",
		KeyType:      nftables.TypeIPAddr,
		KeyByteOrder: binaryutil.BigEndian,
		Interval:     true,
	}

	elements := []nftables.SetElement{{
		Key:         net.ParseIP("0.0.0.0").To4(),
		IntervalEnd: true,
	}}

	for i := range nft.bypassIPv4 {
		bypass := nft.bypassIPv4[i]
		ip := net.ParseIP(bypass)

		if ip != nil {
			elements = append(elements,
				nftables.SetElement{
					Key: ip.To4(),
				},
				nftables.SetElement{
					Key:         nft.nextIP(ip).To4(),
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
				Key:         nft.nextIP(nft.lastIP(cidr).To4()),
				IntervalEnd: true,
			},
		)
	}

	err = conn.AddSet(nft.ipv4BypassSet, elements)
	if err != nil {
		return
	}

	return
}

func (nft *NFTManager) initIPV6BypassSet(conn *nftables.Conn) (err error) {
	nft.ipv6BypassSet = &nftables.Set{
		Table:        nft.table,
		Name:         "bypass6",
		KeyType:      nftables.TypeIP6Addr,
		KeyByteOrder: binaryutil.BigEndian,
		Interval:     true,
	}

	elements := []nftables.SetElement{{
		Key:         net.ParseIP("::").To16(),
		IntervalEnd: true,
	}}

	for i := range nft.bypassIPv6 {
		bypass := nft.bypassIPv6[i]
		ip := net.ParseIP(bypass)
		if ip != nil {
			elements = append(elements,
				nftables.SetElement{
					Key: ip.To16(),
				},
				nftables.SetElement{
					Key:         nft.nextIP(ip.To16()),
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
				Key:         nft.nextIP(nft.lastIP(cidr).To16()),
				IntervalEnd: true,
			},
		)
	}

	err = conn.AddSet(nft.ipv6BypassSet, elements)
	if err != nil {
		return
	}

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

func (nft *NFTManager) initCgroupMap(conn *nftables.Conn) (err error) {
	nft.cgroupMap = &nftables.Set{
		Table:        nft.table,
		Name:         "cgroup-vmap",
		KeyType:      nftables.TypeCGroupV2,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	nft.cgroupMapElement = make(map[string]nftables.SetElement)

	err = conn.AddSet(nft.cgroupMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

func (nft *NFTManager) initMarkMap(conn *nftables.Conn) (err error) {
	nft.markTproxyMap = &nftables.Set{
		Table:        nft.table,
		Name:         "mark-vmap",
		KeyType:      nftables.TypeMark,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	err = conn.AddSet(nft.markTproxyMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

func (nft *NFTManager) initMarkDNSMap(conn *nftables.Conn) (err error) {
	nft.markDNSMap = &nftables.Set{
		Table:        nft.table,
		Name:         "mark-dns-vmap",
		KeyType:      nftables.TypeMark,
		DataType:     nftables.TypeVerdict,
		IsMap:        true,
		KeyByteOrder: binaryutil.NativeEndian,
	}

	err = conn.AddSet(nft.markDNSMap, []nftables.SetElement{})
	if err != nil {
		return
	}

	return
}

func (nft *NFTManager) initOutputMangleChain(conn *nftables.Conn) (err error) {
	// type filter hook prerouting priority mangle; policy accept;
	nft.outputMangleChain = conn.AddChain(&nftables.Chain{
		Table:    nft.table,
		Name:     "output-mangle",
		Type:     nftables.ChainTypeRoute,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &nft.policy,
	})

	err = nft.fillOutputMangleChain(conn, nft.outputMangleChain)
	if err != nil {
		return
	}

	return
}

func (nft *NFTManager) fillOutputMangleChain(
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
			SetID:          nft.ipv4BypassSet.ID,
			SetName:        nft.ipv4BypassSet.Name,
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
			SetID:          nft.ipv6BypassSet.ID,
			SetName:        nft.ipv6BypassSet.Name,
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

func (nft *NFTManager) initOutputNATChain(conn *nftables.Conn) (err error) {
	// type nat hook prerouting priority -100; policy accept;
	nft.outputNATChain = conn.AddChain(&nftables.Chain{
		Table:    nft.table,
		Name:     "output-nat",
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookOutput,
		Priority: nftables.ChainPriorityNATDest,
		Policy:   &nft.policy,
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
			SetName:        nft.markDNSMap.Name,
			SetID:          nft.markDNSMap.ID,
		},
	}
	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: nft.outputNATChain,
		Exprs: exprs,
	})

	return
}

func (nft *NFTManager) initPreroutingChain(conn *nftables.Conn) (err error) {
	// type route hook output priority mangle; policy accept;
	nft.preroutingChain = conn.AddChain(&nftables.Chain{
		Table:    nft.table,
		Name:     "prerouting",
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &nft.policy,
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
			SetID:          nft.ipv4BypassSet.ID,
			SetName:        nft.ipv4BypassSet.Name,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: nft.preroutingChain,
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
			SetID:          nft.ipv6BypassSet.ID,
			SetName:        nft.ipv6BypassSet.Name,
		},
		&expr.Verdict{ // immediate reg 0 return
			Kind: expr.VerdictReturn,
		},
	}

	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: nft.preroutingChain,
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
			SetName:        nft.markTproxyMap.Name,
			SetID:          nft.markTproxyMap.ID,
		},
	}
	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: nft.table,
		Chain: nft.preroutingChain,
		Exprs: exprs,
	})

	return
}
