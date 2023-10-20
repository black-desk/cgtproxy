package nftman

import (
	"errors"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
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

