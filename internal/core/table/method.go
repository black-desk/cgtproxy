package table

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
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
	defer Wrap(&err, "Failed to add cgroup (%s) to nftable.", path)

	Log.Infow("Adding new cgroup to nft.",
		"cgroup", path,
		"target", target,
	)

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(path)
	if err != nil {
		return
	}

	path = t.removeCgroupRoot(path)

	level := uint32(strings.Count(path, "/"))

	switch target.Op {
	case TargetDirect:
		err = t.addBypassCgroupSetIfNeed(level)
		if err != nil {
			return
		}

		// FIXME(black_desk): when add cgroup element to a regular map,
		// we should use NativeEndian.
		setElement := nftables.SetElement{
			Key: binaryutil.NativeEndian.PutUint64(
				fileInfo.Sys().(*syscall.Stat_t).Ino,
			),
		}

		err = t.conn.SetAddElements(
			t.bypassCgroupSets[level].set,
			[]nftables.SetElement{setElement},
		)
		if err != nil {
			return
		}
		err = t.conn.Flush()
		if err != nil {
			return
		}

		t.bypassCgroupSets[level].elements[path] = setElement

	case TargetTProxy:
		err = t.addTProxyCgroupMapIfNeed(level)
		if err != nil {
			return
		}

		// FIXME(black_desk): when add cgroup element to a vmap,
		// it seems that we should use BigEndian. But why?

		setElement := nftables.SetElement{
			Key: binaryutil.BigEndian.PutUint64(
				fileInfo.Sys().(*syscall.Stat_t).Ino,
			),
			VerdictData: &expr.Verdict{
				Kind:  expr.VerdictJump,
				Chain: target.Chain,
			},
		}

		err = t.conn.SetAddElements(
			t.cgroupMaps[level].set,
			[]nftables.SetElement{setElement},
		)
		if err != nil {
			return
		}

		err = t.conn.Flush()
		if err != nil {
			return
		}

		t.cgroupMaps[level].elements[path] = setElement

	case TargetDrop:
		err = t.addTProxyCgroupMapIfNeed(level)
		if err != nil {
			return
		}

		// FIXME(black_desk): when add cgroup element to a vmap,
		// it seems that we should use BigEndian. But why?

		setElement := nftables.SetElement{
			Key: binaryutil.BigEndian.PutUint64(
				fileInfo.Sys().(*syscall.Stat_t).Ino,
			),
			VerdictData: &expr.Verdict{
				Kind: expr.VerdictDrop,
			},
		}

		err = t.conn.SetAddElements(
			t.cgroupMaps[level].set,
			[]nftables.SetElement{setElement},
		)
		if err != nil {
			return
		}
		err = t.conn.Flush()
		if err != nil {
			return
		}

		t.cgroupMaps[level].elements[path] = setElement
	}

	err = t.conn.Flush()
	if err != nil {
		return
	}

	Log.Infow("New cgroup added to nft.",
		"cgroup", path,
	)

	DumpNFTableRules()

	return
}

func (t *Table) addBypassCgroupSetIfNeed(level uint32) (err error) {
	defer Wrap(
		&err,
		"Failed to add bypass cgroup set (level %d) to nftable.",
		level,
	)

	if _, ok := t.bypassCgroupSets[level]; ok {
		return
	}

	DumpNFTableRules()

	Log.Debugw("Adding bypass cgroup set.",
		"level", level,
	)

	set := &nftables.Set{
		Table:   t.table,
		Name:    fmt.Sprintf("bypass-cgroup-%d", level),
		KeyType: nftables.TypeCGroupV2,
	}

	err = t.conn.AddSet(set, []nftables.SetElement{})
	if err != nil {
		return
	}
	t.conn.Flush()
	if err != nil {
		return
	}

	DumpNFTableRules()

	Log.Debugw("Updating prerouting chain.")

	position := len(t.outputRules)

	for i := uint32(0); i < level; i++ {
		if _, ok := t.bypassCgroupSets[i]; !ok {
			break
		}
		position--
	}

	rules := make([]*nftables.Rule, len(t.outputRules[:position]))
	copy(rules, t.outputRules)

	rules = append(rules, &nftables.Rule{
		// socket cgroupv2 level x @bypass-cgroup-x return
		Table: t.table,
		Chain: t.outputChain,
		Exprs: []expr.Any{
			&expr.Socket{ // socket load cgroupv2 => reg 1
				Key:      expr.SocketKeyCgroupv2,
				Level:    uint32(level),
				Register: 1,
			},
			&expr.Lookup{ // lookup reg 1 set bypass-cgroup-x
				SourceRegister: 1,
				SetName:        set.Name,
			},
			&expr.Verdict{ // immediate reg 0 return
				Kind: expr.VerdictReturn,
			},
		},
	})
	rules = append(rules, t.outputRules[position:]...)

	t.conn.FlushChain(t.outputChain)
	t.conn.AddSet(t.protoSet, t.protoSetElement)
	for _, rule := range rules {
		t.conn.AddRule(rule)
	}

	err = t.conn.Flush()
	if err != nil {
		return
	}

	t.outputRules = rules

	t.bypassCgroupSets[level] = cgroupSet{
		set:      set,
		elements: map[string]nftables.SetElement{},
	}

	return
}

func (t *Table) addTProxyCgroupMapIfNeed(level uint32) (err error) {
	defer Wrap(
		&err,
		"Failed to add tproxy cgroup set (level %d) to nftable.",
		level,
	)

	if _, ok := t.cgroupMaps[level]; ok {
		return
	}

	DumpNFTableRules()

	Log.Debugw("Adding tproxy cgroup map.",
		"level", level,
	)

	set := &nftables.Set{
		Table:    t.table,
		Name:     fmt.Sprintf("cgroup-map-%d", level),
		KeyType:  nftables.TypeCGroupV2,
		DataType: nftables.TypeVerdict,
		IsMap:    true,
	}

	err = t.conn.AddSet(set, []nftables.SetElement{})
	if err != nil {
		return
	}
	err = t.conn.Flush()
	if err != nil {
		return
	}

	DumpNFTableRules()

	Log.Debugw("Updating prerouting chain.")

	position := len(t.preroutingRules) - 1

	for i := uint32(0); i < level; i++ {
		if _, ok := t.cgroupMaps[i]; !ok {
			break
		}
		position--
	}

	rules := make([]*nftables.Rule, len(t.preroutingRules[:position]))

	copy(rules, t.preroutingRules)

	rules = append(rules, &nftables.Rule{
		// socket cgroupv2 level x vmap @cgroup-map-x
		Table: t.table,
		Chain: t.preroutingChain,
		Exprs: []expr.Any{
			&expr.Socket{ // socket load cgroupv2 => reg 1
				Key:      expr.SocketKeyCgroupv2,
				Level:    uint32(level),
				Register: 1,
			},
			&expr.Lookup{ // lookup reg 1 set cgroup-map-x dreg 0
				SourceRegister: 1,
				IsDestRegSet:   true,
				SetName:        set.Name,
			},
		},
	})
	rules = append(rules, t.preroutingRules[position:]...)

	t.conn.FlushChain(t.preroutingChain)
	t.conn.AddSet(t.protoSet, t.protoSetElement)
	for _, rule := range rules {
		t.conn.AddRule(rule)
	}
	err = t.conn.Flush()
	if err != nil {
		return
	}

	t.preroutingRules = rules

	t.cgroupMaps[level] = cgroupSet{
		set:      set,
		elements: map[string]nftables.SetElement{},
	}

	return
}

func (t *Table) RemoveCgroup(path string) (err error) {
	defer Wrap(
		&err,
		"Failed to remove cgroup (%s) from nftable.",
		path,
	)

	path = t.removeCgroupRoot(path)

	level := uint32(strings.Count(path, "/"))

	if _, ok := t.bypassCgroupSets[level].elements[path]; ok {
		Log.Infow("Removing bypass rule from nft for this cgroup.",
			"cgroup", path,
		)

		t.conn.SetDeleteElements(
			t.bypassCgroupSets[level].set,
			[]nftables.SetElement{
				t.bypassCgroupSets[level].elements[path],
			},
		)
		if err != nil {
			return
		}
		err = t.conn.Flush()
		if err != nil {
			return
		}

		delete(t.bypassCgroupSets[level].elements, path)
	} else if _, ok := t.cgroupMaps[level].elements[path]; ok {
		Log.Infow("Removing proxy rule from nft for this cgroup.",
			"cgroup", path,
		)

		err = t.conn.SetDeleteElements(
			t.cgroupMaps[level].set,
			[]nftables.SetElement{
				t.cgroupMaps[level].elements[path],
			},
		)
		if err != nil {
			return
		}
		err = t.conn.Flush()
		if err != nil {
			return
		}

		delete(t.cgroupMaps[level].elements, path)
	} else {
		Log.Debugw("Nothing to do with this cgroup",
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

	Log.Debugw("Adding chain and rules for tproxy.",
		"tproxy", tp,
	)

	chain := &nftables.Chain{
		Table: t.table,
		Name:  tp.Name,
	}
	t.tproxyChains = append(t.tproxyChains, chain)
	t.conn.AddChain(chain)
	err = t.conn.Flush()
	if err != nil {
		return
	}

	tproxy := &expr.TProxy{ // tproxy port reg 1
		Family:  byte(nftables.TableFamilyUnspecified),
		RegPort: 1,
	}

	err = t.conn.AddSet(t.protoSet, t.protoSetElement)
	if err != nil {
		return
	}

	rule := &nftables.Rule{
		// meta l4proto { tcp, udp } tproxy to ...
		Table: t.table,
		Chain: chain,
		Exprs: []expr.Any{
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
		},
	}

	lookup := &rule.Exprs[1]

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

	t.conn.AddRule(rule)

	err = t.conn.Flush()
	if err != nil {
		return
	}

	Log.Debug("chain and rules added for this tproxy.")

	DumpNFTableRules()

	return
}

func (t *Table) Clear() (err error) {
	defer Wrap(&err, "Error occurs while removing nftable.")

	t.conn.DelTable(t.table)
	err = t.conn.Flush()
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

func (t *Table) getRules(
	tb *nftables.Table, ch *nftables.Chain,
) (
	ret []*nftables.Rule,
) {
	// FIXME(black_desk): (nftables.Conn).GetRules has a bug.
	// It does not recognize a nftable event has NFT_MSG_NEWGEN in header.
	// And return an error in this situation.
	// According to `nft` source, when this kind of event arrive,
	// it set errno to EINTR,
	// which properly means that caller should have a retry.
	// Check https://git.netfilter.org/nftables/tree/src/mnl.c?id=6ab0fd6c67dbccedb49209b94eb7f740dd32fd2a#n169

	var err error
	for {
		ret, err = t.conn.GetRules(tb, ch)
		if err == nil {
			break
		}
	}
	return
}
