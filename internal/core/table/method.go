package table

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"golang.org/x/sys/unix"
)

type TargetOp uint32

const (
	TargetDrop TargetOp = iota
	TargetTProxy
	TargetDirect
)

type Target struct {
	Op    TargetOp
	Chain string
}

func (t *Table) AddCgroup(path string, target *Target) (err error) {
	defer Wrap(&err, "Failed to add cgroup (%s) to nftable.", path)

	path = filepath.Clean(path)[len(t.cgroupRoot):]

	level := uint32(strings.Count(path, "/"))

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(path)
	if err != nil {
		return
	}

	setElement := nftables.SetElement{
		Key: binaryutil.NativeEndian.PutUint64(
			fileInfo.Sys().(*syscall.Stat_t).Ino,
		),
	}

	switch target.Op {
	case TargetDirect:
		err = t.addBypassCgroupSetIfNeed(level)
		if err != nil {
			return
		}

		if err = t.conn.SetAddElements(
			t.bypassCgroupSets[level].set,
			[]nftables.SetElement{setElement},
		); err != nil {
			return
		}

		t.bypassCgroupSets[level].elements[path] = setElement

	case TargetTProxy:
		err = t.addTProxyCgroupMapIfNeed(level)
		if err != nil {
			return
		}

		setElement.VerdictData = &expr.Verdict{
			Kind:  expr.VerdictJump,
			Chain: target.Chain,
		}

		if err = t.conn.SetAddElements(
			t.cgroupMaps[level].set,
			[]nftables.SetElement{setElement},
		); err != nil {
			return
		}

		t.cgroupMaps[level].elements[path] = setElement

	case TargetDrop:
		err = t.addTProxyCgroupMapIfNeed(level)
		if err != nil {
			return
		}

		setElement.VerdictData = &expr.Verdict{
			Kind: expr.VerdictDrop,
		}

		if err = t.conn.SetAddElements(
			t.cgroupMaps[level].set,
			[]nftables.SetElement{setElement},
		); err != nil {
			return
		}

		t.cgroupMaps[level].elements[path] = setElement
	}

	err = t.conn.Flush()
	if err != nil {
		return
	}

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

	set := &nftables.Set{
		Table:   t.table,
		Name:    fmt.Sprintf("bypass-cgroup-%d", level),
		KeyType: nftables.TypeCGroupV2,
	}

	err = t.conn.AddSet(set, []nftables.SetElement{})
	if err != nil {
		return
	}

	var position uint64

	for i := level - 1; i >= 0; i-- {
		if i == 0 {
			position = consts.RuleInsertHandle
			break
		}

		if _, ok := t.bypassCgroupSets[i]; !ok {
			continue
		}

		position = uint64(i)
		break
	}

	// WARN(black_desk): Seems InsertRule will not insert rule but
	// will replace rule with Handle is set.
	// Remember to update comment in `addBypassCgroupSetIfNeed` as well.

	// socket cgroupv2 level x @bypass-cgroup-x return # handle x
	t.conn.InsertRule(&nftables.Rule{
		Table:    t.table,
		Chain:    t.outputChain,
		Handle:   uint64(level),
		Position: position,
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

	t.bypassCgroupSets[level] = cgroupSet{
		set: set,
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

	var position uint64

	for i := level - 1; i >= 0; i-- {
		if i == 0 {
			position = consts.RuleInsertHandle
			break
		}

		if _, ok := t.cgroupMaps[i]; !ok {
			continue
		}

		position = uint64(i)
		break
	}

	// WARN(black_desk): Same as `addBypassCgroupSetIfNeed`

	// socket cgroupv2 level x vmap @cgroup-map-x # handle x
	t.conn.InsertRule(&nftables.Rule{
		Table:    t.table,
		Chain:    t.preroutingChain,
		Handle:   uint64(level),
		Position: position,
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

	t.cgroupMaps[level] = cgroupSet{
		set: set,
	}

	return
}

func (t *Table) RemoveCgroup(path string) (err error) {
	defer Wrap(
		&err,
		"Failed to remove cgroup (%s) from nftable.",
		path,
	)

	path = filepath.Clean(path)[len(t.cgroupRoot):]

	level := uint32(strings.Count(path, "/"))

	if _, ok := t.bypassCgroupSets[level].elements[path]; ok {
		if err = t.conn.SetDeleteElements(
			t.bypassCgroupSets[level].set,
			[]nftables.SetElement{
				t.bypassCgroupSets[level].elements[path],
			},
		); err != nil {
			return
		}

		delete(t.bypassCgroupSets[level].elements, path)
	} else if _, ok := t.cgroupMaps[level].elements[path]; ok {
		if err = t.conn.SetDeleteElements(
			t.cgroupMaps[level].set,
			[]nftables.SetElement{
				t.cgroupMaps[level].elements[path],
			},
		); err != nil {
			return
		}

		delete(t.cgroupMaps[level].elements, path)
	}

	return
}

func (t *Table) AddChainAndRulesForTProxy(tp *config.TProxy) (name string) {
	// type filter hook prerouting priority mangle; policy accept;
	chain := &nftables.Chain{
		Table:    t.table,
		Name:     tp.Name,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookPrerouting,
		Priority: nftables.ChainPriorityMangle,
		Policy:   &t.policy,
	}
	t.tproxyChains = append(t.tproxyChains, chain)

	tproxy := &expr.TProxy{ // tproxy port reg 1
		Family:      byte(nftables.TableFamilyINet),
		TableFamily: byte(nftables.TableFamilyINet),
		RegPort:     1,
	}

	rule := &nftables.Rule{
		// meta l4proto tcp tproxy to ...
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
			},
			&expr.Immediate{ // immediate reg 1 ...
				Register: 1,
				Data:     binaryutil.NativeEndian.PutUint16(tp.Port),
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

	t.tproxyRules[chain.Name] = append(t.tproxyRules[chain.Name], rule)

	name = chain.Name
	return
}

func (t *Table) FlushInitialContent() (err error) {
	defer Wrap(&err, "Error occurs while flushing nftable.")

	t.conn.AddTable(t.table)
	t.conn.AddSet(t.ipv4BypassSet, t.ipv4BypassSetElement)
	t.conn.AddSet(t.ipv6BypassSet, t.ipv6BypassSetElement)

	for i := range t.tproxyChains {
		if t.tproxyChains[i] == nil {
			return
		}
		t.conn.AddChain(t.tproxyChains[i])

		chain := t.tproxyChains[i].Name

		if t.tproxyRules[chain] == nil {
			return
		}

		for i := range t.tproxyRules[chain] {
			if t.tproxyRules[chain][i] == nil {
				return
			}
			t.conn.AddRule(t.tproxyRules[chain][i])
		}
	}

	t.conn.AddChain(t.outputChain)
	for i := range t.outputRules {
		if t.outputRules[i] == nil {
			return
		}
		t.conn.AddRule(t.outputRules[i])
	}

	t.conn.AddChain(t.preroutingChain)
	for i := range t.preroutingRules {
		if t.preroutingRules[i] == nil {
			return
		}
		t.conn.AddRule(t.preroutingRules[i])
	}

	err = t.conn.Flush()
	return
}

func (t *Table) Clear() (err error) {
	defer Wrap(&err, "Error occurs while removing nftable.")

	t.conn.DelTable(t.table)
	err = t.conn.Flush()
	return
}
