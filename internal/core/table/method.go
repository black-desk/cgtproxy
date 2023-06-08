package table

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/black-desk/cgtproxy/internal/config"
	. "github.com/black-desk/cgtproxy/internal/log"
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

	inode := fileInfo.Sys().(*syscall.Stat_t).Ino

	path = t.removeCgroupRoot(path)

	Log.Debugw("Get inode of cgroup file using stat(2).",
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

	err = t.conn.SetAddElements(
		t.cgroupMap,
		[]nftables.SetElement{setElement},
	)
	if err != nil {
		return
	}
	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
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

	Log.Debugw("Existing levels.",
		"levels", levels,
	)

	Log.Debugw("Flushing output chain.")

	t.conn.FlushChain(t.outputChain)
	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	Log.Debugw("Fill output chain again.")

	err = t.fillOutputChain()
	if err != nil {
		return
	}

	for i := len(levels) - 1; i >= 0; i-- {
		err = t.addCgroupRuleForLevel(levels[i])
		if err != nil {
			return
		}
	}

	Log.Infow("New cgroup added to nft.",
		"cgroup", path,
	)

	DumpNFTableRules()

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
		Log.Infow("Removing bypass rule from nft for this cgroup.",
			"cgroup", path,
		)

		t.conn.SetDeleteElements(
			t.cgroupMap,
			[]nftables.SetElement{
				t.cgroupMapElement[path],
			},
		)
		if err != nil {
			return
		}
		err = t.conn.Flush()
		ignoreNoBufferSpaceAvailable(&err)
		if err != nil {
			return
		}

		delete(t.cgroupMapElement, path)

		DumpNFTableRules()
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

	{ // -MARK chain

		chain := &nftables.Chain{
			Table: t.table,
			Name:  tp.Name + "-MARK",
		}

		t.conn.AddChain(chain)
		err = t.conn.Flush()
		ignoreNoBufferSpaceAvailable(&err)
		if err != nil {
			return
		}

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

		t.conn.AddRule(&nftables.Rule{
			Table: t.table,
			Chain: chain,
			Exprs: exprs,
		})

		err = t.conn.Flush()
		ignoreNoBufferSpaceAvailable(&err)
		if err != nil {
			return
		}
	}

	{ // tproxy chain
		chain := &nftables.Chain{
			Table: t.table,
			Name:  tp.Name,
		}

		t.conn.AddChain(chain)
		err = t.conn.Flush()
		ignoreNoBufferSpaceAvailable(&err)
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

		t.conn.AddRule(rule)
		err = t.conn.Flush()
		ignoreNoBufferSpaceAvailable(&err)
		if err != nil {
			return
		}
	}

	{
		setElement := nftables.SetElement{
			Key: binaryutil.NativeEndian.PutUint32(uint32(tp.Mark)),
			VerdictData: &expr.Verdict{
				Kind:  expr.VerdictGoto,
				Chain: tp.Name,
			},
		}
		err = t.conn.SetAddElements(t.markMap, []nftables.SetElement{setElement})
		if err != nil {
			return
		}
		err = t.conn.Flush()
		ignoreNoBufferSpaceAvailable(&err)
		if err != nil {
			return
		}
	}

	Log.Debug("Nftable chain added for this tproxy.",
		"tproxy", tp,
	)

	DumpNFTableRules()

	return
}

func (t *Table) Clear() (err error) {
	defer Wrap(&err, "Error occurs while removing nftable.")

	t.conn.DelTable(t.table)
	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	DumpNFTableRules()
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

func (t *Table) addCgroupRuleForLevel(level int) (err error) {
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
		Chain: t.outputChain,
		Exprs: exprs,
	}

	t.conn.AddRule(rule)
	err = t.conn.Flush()
	ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	return
}
