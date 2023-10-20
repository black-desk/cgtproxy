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

func (nft *NFTManager) AddCgroup(path string, target *Target) (err error) {
	defer Wrap(&err, "add cgroup (%s) to nftable", path)

	nft.log.Infow("Adding new cgroup to nft.",
		"cgroup", path,
		"target", target,
	)

	if _, ok := nft.cgroupMapElement[path]; ok {
		return os.ErrExist
	}

	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(path)
	if err != nil {
		return
	}

	inode := fileInfo.Sys().(*syscall.Stat_t).Ino

	path = nft.removeCgroupRoot(path)

	nft.log.Debugw("Get inode of cgroup file using stat(2).",
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
		nft.cgroupMap,
		[]nftables.SetElement{setElement},
	)
	if err != nil {
		return
	}

	nft.cgroupMapElement[path] = setElement

	tmp := map[int]struct{}{}

	for path := range nft.cgroupMapElement {
		level := strings.Count(path, "/")
		tmp[level] = struct{}{}
	}

	levels := make([]int, 0, len(tmp))
	for level := range tmp {
		levels = append(levels, level)
	}
	sort.Ints(levels)

	nft.log.Debugw("Existing levels.",
		"levels", levels,
	)

	conn.FlushChain(nft.outputMangleChain)

	err = nft.fillOutputMangleChain(conn, nft.outputMangleChain)
	if err != nil {
		return
	}

	nft.log.Debugw("Output chain refilled.")
	nft.dumpNFTableRules()

	for i := len(levels) - 1; i >= 0; i-- {
		err = nft.addCgroupRuleForLevel(conn, levels[i])
		if err != nil {
			return
		}
	}

	err = conn.Flush()
	nft.ignoreNoBufferSpaceAvailable(&err)
	if err != nil {
		return
	}

	nft.log.Infow("New cgroup added to nft.",
		"cgroup", path,
	)

	nft.dumpNFTableRules()

	return
}

func (nft *NFTManager) RemoveCgroup(path string) (err error) {
	defer Wrap(
		&err,
		"Failed to remove cgroup (%s) from nftable.",
		path,
	)

	path = nft.removeCgroupRoot(path)

	if _, ok := nft.cgroupMapElement[path]; ok {
		nft.log.Infow("Removing rule from nft for this cgroup.",
			"cgroup", path,
		)

		var conn *nftables.Conn
		conn, err = nftables.New()
		if err != nil {
			return
		}

		conn.SetDeleteElements(
			nft.cgroupMap,
			[]nftables.SetElement{
				nft.cgroupMapElement[path],
			},
		)
		if err != nil {
			return
		}

		err = conn.Flush()
		nft.ignoreNoBufferSpaceAvailable(&err)
		if err != nil {
			return
		}

		delete(nft.cgroupMapElement, path)

		nft.dumpNFTableRules()
	} else {
		nft.log.Debugw("Nothing to do with this cgroup",
			"cgroup", path,
		)
	}

	return
}

func (nft *NFTManager) AddChainAndRulesForTProxy(tp *config.TProxy) (err error) {
	defer Wrap(
		&err,
		"Failed to add chain and rules to nft table for tproxy: %#v",
		tp,
	)

	nft.log.Debugw("Adding chain and rules for tproxy.",
		"tproxy", tp,
	)

	var conn *nftables.Conn
	conn, err = nftables.New()
	if err != nil {
		return
	}

	_, err = nft.addMarkChainForTProxy(conn, tp)
	if err != nil {
		return
	}

	var chain *nftables.Chain

	chain, err = nft.addTproxyChainForTProxy(conn, tp)
	if err != nil {
		return
	}

	err = nft.updateMarkTproxyMap(conn, tp.Mark, chain.Name)
	if err != nil {
		return
	}

	if tp.DNSHijack != nil {
		chain, err = nft.addDNSChainForTproxy(conn, tp)
		if err != nil {
			return
		}

		err = nft.updateMarkDNSMap(conn, tp.Mark, chain.Name)
		if err != nil {
			return
		}
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	nft.log.Debug("Nftable chain added for this tproxy.",
		"tproxy", tp,
	)

	nft.dumpNFTableRules()

	return
}

func (nft *NFTManager) Clear() (err error) {
	defer Wrap(&err, "remove nftable.")

	var conn *nftables.Conn
	conn, err = nftables.New()
	if err != nil {
		return
	}

	conn.DelTable(nft.table)
	err = conn.Flush()
	nft.ignoreNoBufferSpaceAvailable(&err)
	if errors.Is(err, os.ErrNotExist) {
		nft.log.Debugw("Table not exist, nothing to remove.",
			"table", nft.table.Name,
		)
		err = nil
	} else if err != nil {
		return
	}

	nft.dumpNFTableRules()
	return
}
