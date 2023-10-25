package nftman

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

func (nft *NFTManager) nextIP(ip net.IP) (ret net.IP) {
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

	nft.log.Debugf("Next IP of %s is %s", ip.String(), next.String())

	ret = next
	return
}

func (nft *NFTManager) lastIP(ipnet *net.IPNet) (ret net.IP) {
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)

	for i := range ip {
		ip[i] |= ^ipnet.Mask[i]
	}

	nft.log.Debugf("Last IP in net %s is %s", ipnet.String(), ip.String())

	ret = ip
	return
}

func (t *NFTManager) addMarkChainForTProxy(
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

func (t *NFTManager) addTproxyChainForTProxy(
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
	} else {
		// NOTE:
		// Only add set when we use it, otherwise we will get an EINVAL
		// https://github.com/torvalds/linux/blob/4f82870119a46b0d04d91ef4697ac4977a255a9d/net/netfilter/nf_tables_api.c#L9881

		err = conn.AddSet(t.protoSet, t.protoSetElement)
		if err != nil {
			return
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

func (t *NFTManager) updateMarkTproxyMap(
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

func (t *NFTManager) updateMarkDNSMap(
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

func (t *NFTManager) addDNSChainForTproxy(
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

func (t *NFTManager) removeCgroupRootFromPath(path string) string {
	path = filepath.Clean(path)
	if strings.HasPrefix(path, string(t.cgroupRoot)) {
		path = path[len(t.cgroupRoot):]
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

func (t *NFTManager) addCgroupRuleForLevel(
	conn *nftables.Conn, level int,
) (
	err error,
) {
	defer Wrap(&err,
		"update output chain for level %d cgroup", level)

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
	return
}

func getNFTableRules() string {
	out, err := exec.Command("nft", "list", "ruleset").Output()
	if err != nil {
		panic(err)
	}

	return string(out)
}

func (nft *NFTManager) genSetElement(route *types.Route) (ret nftables.SetElement, err error) {
	defer Wrap(&err, "generating set element for route",
		"Path", route.Path,
		"Target", route.Target,
	)

	nft.log.Debugw("Generating set element for new cgroup route.",
		"Path", route.Path,
		"Target", route.Target,
	)

	path := route.Path
	target := route.Target

	if _, ok := nft.cgroupMapElement[path]; ok {
		err = os.ErrExist
		return
	}

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(path)
	if err != nil {
		return
	}

	inode := fileInfo.Sys().(*syscall.Stat_t).Ino

	route.Path = nft.removeCgroupRootFromPath(path)
	path = route.Path

	nft.log.Debugw("Get inode of cgroup file using stat(2).",
		"path", path,
		"inode", inode,
	)

	setElement := nftables.SetElement{
		Key:         binaryutil.NativeEndian.PutUint64(inode),
		VerdictData: nil,
	}

	switch target.Op {
	case types.TargetDirect:
		setElement.VerdictData = &expr.Verdict{
			Kind: expr.VerdictReturn,
		}

	case types.TargetTProxy:
		setElement.VerdictData = &expr.Verdict{
			Kind:  expr.VerdictGoto,
			Chain: target.Chain,
		}

	case types.TargetDrop:
		setElement.VerdictData = &expr.Verdict{
			Kind: expr.VerdictDrop,
		}
	}

	ret = setElement
	return
}
