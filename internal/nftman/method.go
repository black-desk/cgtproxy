package nftman

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

type TargetOp uint32

const (
	TargetNoop TargetOp = iota
	TargetDrop
	TargetTProxy
	TargetDirect
)

type Target struct {
	Op    TargetOp
	Chain string
}

func (t *Table) AddCgroup(path string, target *Target) (err error) {
	defer Wrap(&err, "add cgroup (%s) to nftable", path)

	t.log.Infow("Adding new cgroup to nft.",
		"cgroup", path,
		"target", target,
	)

	if _, ok := t.cgroupMapElement[path]; ok {
		return os.ErrExist
	}

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(path)
	if err != nil {
		return
	}

	inode := fileInfo.Sys().(*syscall.Stat_t).Ino

	path = t.removeCgroupRoot(path)

	t.log.Debugw("Get inode of cgroup file using stat(2).",
		"path", path,
		"inode", inode,
	)

	setElement := nftables.SetElement{
		Key:         binaryutil.NativeEndian.PutUint64(inode),
		VerdictData: nil,
	}

	switch target.Op {
	case TargetDirect:
		setElement.VerdictData = &expr.Verdict{
			Kind: expr.VerdictReturn,
		}

	case TargetTProxy:
		setElement.VerdictData = &expr.Verdict{
			Kind:  expr.VerdictGoto,
			Chain: target.Chain,
		}

	case TargetDrop:
		setElement.VerdictData = &expr.Verdict{
			Kind: expr.VerdictDrop,
		}
	}

	var conn *nftables.Conn
	conn, err = nftables.New()
	if err != nil {
		return
	}

	err = conn.SetAddElements(
		t.cgroupMap,
		[]nftables.SetElement{setElement},
	)
	if err != nil {
		return
	}

	t.cgroupMapElement[path] = setElement

	tmp := map[int]struct{}{}

	for path := range t.cgroupMapElement {
		level := strings.Count(path, "/")
		tmp[level] = struct{}{}
	}

	levels := make([]int, 0, len(tmp))
	for level := range tmp {
		levels = append(levels, level)
	}
	sort.Ints(levels)

	t.log.Debugw("Existing levels.",
		"levels", levels,
	)

	conn.FlushChain(t.outputMangleChain)

	err = t.fillOutputMangleChain(conn, t.outputMangleChain)
	if err != nil {
		return
	}

	t.log.Debugw("Output chain refilled.")
	t.DumpNFTableRules()

	for i := len(levels) - 1; i >= 0; i-- {
		err = t.addCgroupRuleForLevel(conn, levels[i])
		if err != nil {
			return
		}
	}

	err = conn.Flush()
	t.ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	t.log.Infow("New cgroup added to nft.",
		"cgroup", path,
	)

	t.DumpNFTableRules()

	return
}

func (t *Table) RemoveCgroup(path string) (err error) {
	defer Wrap(
		&err,
		"Failed to remove cgroup (%s) from nftable.",
		path,
	)

	path = t.removeCgroupRoot(path)

	if _, ok := t.cgroupMapElement[path]; ok {
		t.log.Infow("Removing rule from nft for this cgroup.",
			"cgroup", path,
		)

		var conn *nftables.Conn
		conn, err = nftables.New()
		if err != nil {
			return
		}

		conn.SetDeleteElements(
			t.cgroupMap,
			[]nftables.SetElement{
				t.cgroupMapElement[path],
			},
		)
		if err != nil {
			return
		}

		err = conn.Flush()
		t.ignoreNoBufferSpaceAvailable(&err)
		if err != nil {
			return
		}

		delete(t.cgroupMapElement, path)

		t.DumpNFTableRules()
	} else {
		t.log.Debugw("Nothing to do with this cgroup",
			"cgroup", path,
		)
	}

	return
}

func (t *Table) AddChainAndRulesForTProxy(tp *config.TProxy) (err error) {
	defer Wrap(
		&err,
		"Failed to add chain and rules to nft table for tproxy: %#v",
		tp,
	)

	t.log.Debugw("Adding chain and rules for tproxy.",
		"tproxy", tp,
	)

	var conn *nftables.Conn
	conn, err = nftables.New()
	if err != nil {
		return
	}

	_, err = t.addMarkChainForTProxy(conn, tp)
	if err != nil {
		return
	}

	var chain *nftables.Chain

	chain, err = t.addTproxyChainForTProxy(conn, tp)
	if err != nil {
		return
	}

	err = t.updateMarkTproxyMap(conn, tp.Mark, chain.Name)
	if err != nil {
		return
	}

	if tp.DNSHijack != nil {
		chain, err = t.addDNSChainForTproxy(conn, tp)
		if err != nil {
			return
		}

		err = t.updateMarkDNSMap(conn, tp.Mark, chain.Name)
		if err != nil {
			return
		}
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	t.log.Debug("Nftable chain added for this tproxy.",
		"tproxy", tp,
	)

	t.DumpNFTableRules()

	return
}

func (t *Table) addMarkChainForTProxy(
	conn *nftables.Conn, tp *config.TProxy,
) (
	ret *nftables.Chain, err error,
) {
	chain := &nftables.Chain{
		Table: t.table,
		Name:  tp.Name + "-MARK",
	}

	conn.AddChain(chain)

	// meta mark set ...

	exprs := []expr.Any{
		&expr.Immediate{ // immediate reg 1 ...
			Register: 1,
			Data: binaryutil.NativeEndian.PutUint32(
				uint32(tp.Mark),
			),
		},
		&expr.Meta{
			Key:            expr.MetaKeyMARK,
			SourceRegister: true,
			Register:       1,
		},
	}

	exprs = addDebugCounter(exprs)

	conn.AddRule(&nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	})

	ret = chain

	return
}

func (t *Table) addTproxyChainForTProxy(
	conn *nftables.Conn, tp *config.TProxy,
) (
	ret *nftables.Chain, err error,
) {
	chain := &nftables.Chain{
		Table: t.table,
		Name:  tp.Name,
	}

	conn.AddChain(chain)

	tproxy := &expr.TProxy{ // tproxy port reg 1
		Family:  byte(nftables.TableFamilyUnspecified),
		RegPort: 1,
	}

	err = conn.AddSet(t.protoSet, t.protoSetElement)
	if err != nil {
		return
	}

	exprs := []expr.Any{
		&expr.Meta{ // meta load l4proto => reg 1
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set __set%d
			SourceRegister: 1,
			SetID:          t.protoSet.ID,
			SetName:        t.protoSet.Name,
		},
		&expr.Immediate{ // immediate reg 1 ...
			Register: 1,
			Data:     binaryutil.BigEndian.PutUint16(tp.Port),
		},
		tproxy,
	}

	lookup := &exprs[1]

	if tp.NoUDP {
		*lookup = &expr.Cmp{ // cmp eq reg 1 0x00000006
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.IPPROTO_TCP},
		}
	}

	if tp.NoIPv6 {
		tproxy.Family = byte(nftables.TableFamilyIPv4)
	}

	exprs = addDebugCounter(exprs)

	rule := &nftables.Rule{
		// meta l4proto { tcp, udp } tproxy to ...
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	}

	conn.AddRule(rule)

	ret = chain

	return
}

func (t *Table) updateMarkTproxyMap(
	conn *nftables.Conn, mark config.FireWallMark, chain string,
) (
	err error,
) {
	setElement := nftables.SetElement{
		Key: binaryutil.NativeEndian.PutUint32(uint32(mark)),
		VerdictData: &expr.Verdict{
			Kind:  expr.VerdictGoto,
			Chain: chain,
		},
	}
	err = conn.SetAddElements(
		t.markTproxyMap,
		[]nftables.SetElement{setElement},
	)
	if err != nil {
		return
	}

	return
}

func (t *Table) updateMarkDNSMap(
	conn *nftables.Conn, mark config.FireWallMark, chain string,
) (
	err error,
) {
	setElement := nftables.SetElement{
		Key: binaryutil.NativeEndian.PutUint32(uint32(mark)),
		VerdictData: &expr.Verdict{
			Kind:  expr.VerdictGoto,
			Chain: chain,
		},
	}
	err = conn.SetAddElements(
		t.markDNSMap,
		[]nftables.SetElement{setElement},
	)
	if err != nil {
		return
	}

	return
}

func (t *Table) addDNSChainForTproxy(
	conn *nftables.Conn, tp *config.TProxy,
) (
	ret *nftables.Chain, err error,
) {
	chain := &nftables.Chain{
		Table: t.table,
		Name:  tp.Name + "-DNS",
	}

	conn.AddChain(chain)

	defer func() {
		if err != nil {
			return
		}

		ret = chain
	}()

	exprs := []expr.Any{
		&expr.Meta{ // meta load l4proto => reg 1
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00000011
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.IPPROTO_UDP},
		},
		&expr.Payload{ // payload load 2b @ transport header + 2 => reg 1
			OperationType: expr.PayloadLoad,
			DestRegister:  1,
			Base:          expr.PayloadBaseTransportHeader,
			Offset:        2,
			Len:           2,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00003500
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     binaryutil.BigEndian.PutUint16(53),
		},
		&expr.Immediate{ // immediate reg 1 xxx
			Register: 1,
			Data:     net.ParseIP(*tp.DNSHijack.IP).To4(),
		},
		&expr.Immediate{ // immediate reg 2 xxx
			Register: 2,
			Data:     binaryutil.BigEndian.PutUint16(tp.DNSHijack.Port),
		},
		&expr.NAT{ // nat dnat ip addr_min reg 1
			Type:        expr.NATTypeDestNAT,
			Family:      unix.NFPROTO_IPV4,
			RegAddrMin:  1,
			RegProtoMin: 2,
		},
	}
	rule := &nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	}

	conn.AddRule(rule)

	if !tp.DNSHijack.TCP {
		return
	}

	exprs = []expr.Any{
		&expr.Meta{ // meta load l4proto => reg 1
			Key:      expr.MetaKeyL4PROTO,
			Register: 1,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00000006
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     []byte{unix.IPPROTO_TCP},
		},
		&expr.Payload{ // payload load 2b @ transport header + 2 => reg 1
			OperationType:  expr.PayloadLoad,
			DestRegister:   1,
			SourceRegister: 0,
			Base:           expr.PayloadBaseTransportHeader,
			Offset:         2,
			Len:            2,
		},
		&expr.Cmp{ // cmp eq reg 1 0x00003500
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     binaryutil.BigEndian.PutUint16(53),
		},
		&expr.Immediate{ // immediate reg 1 xxx
			Register: 1,
			Data:     net.ParseIP(*tp.DNSHijack.IP).To4(),
		},
		&expr.Immediate{ // immediate reg 2 xxx
			Register: 2,
			Data:     binaryutil.BigEndian.PutUint16(tp.DNSHijack.Port),
		},
		&expr.NAT{ // nat dnat ip addr_min reg 1
			Type:        expr.NATTypeDestNAT,
			Family:      unix.NFPROTO_IPV4,
			RegAddrMin:  1,
			RegProtoMin: 2,
		},
	}
	rule = &nftables.Rule{
		Table: t.table,
		Chain: chain,
		Exprs: exprs,
	}

	conn.AddRule(rule)

	return
}

func (t *Table) Clear() (err error) {
	defer Wrap(&err, "remove nftable.")

	var conn *nftables.Conn
	conn, err = nftables.New()
	if err != nil {
		return
	}

	conn.DelTable(t.table)
	err = conn.Flush()
	t.ignoreNoBufferSpaceAvailable(&err)
	if errors.Is(err, os.ErrNotExist) {
		t.log.Debugw("Table not exist, nothing to remove.",
			"table", t.table.Name,
		)
		err = nil
	} else if err != nil {
		return
	}

	t.DumpNFTableRules()
	return
}

func (t *Table) removeCgroupRoot(path string) string {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, string(t.cgroupRoot)) {
		path = path[len(t.cgroupRoot):]
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

func (t *Table) addCgroupRuleForLevel(
	conn *nftables.Conn, level int,
) (
	err error,
) {
	defer Wrap(&err,
		"Failed to update output chain for level %d cgroup.", level)

	exprs := []expr.Any{
		&expr.Socket{ // socket load cgroupv2 => reg 1
			Key:      expr.SocketKeyCgroupv2,
			Level:    uint32(level),
			Register: 1,
		},
		&expr.Lookup{ // lookup reg 1 set cgroup-map dreg 0
			SourceRegister: 1,
			IsDestRegSet:   true,
			SetName:        t.cgroupMap.Name,
			SetID:          t.cgroupMap.ID,
		},
	}

	exprs = addDebugCounter(exprs)

	rule := &nftables.Rule{
		Table: t.table,
		Chain: t.outputMangleChain,
		Exprs: exprs,
	}

	conn.AddRule(rule)
	err = conn.Flush()
	t.ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}
