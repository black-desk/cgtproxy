package nftman

import (
	"errors"
	"sort"
	"strings"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
)

func (nft *NFTManager) AddRoutes(routes []types.Route) (err error) {
	defer Wrap(&err, "add %d routes to nftable", len(routes))

	var conn *nftables.Conn
	conn, err = nft.connector.Connect()
	if err != nil {
		return
	}
	elements := []nftables.SetElement{}

	tmpCGroupMapElement := make(map[string]nftables.SetElement, len(nft.cgroupMapElement))
	for k, v := range nft.cgroupMapElement {
		tmpCGroupMapElement[k] = v
	}

	nft.log.Debugw("old cgroup map elements", "value", tmpCGroupMapElement)

	for i := range routes {
		var element nftables.SetElement
		element, err = nft.genSetElement(&routes[i])
		if err != nil {
			return
		}
		elements = append(elements, element)
		tmpCGroupMapElement[routes[i].Path] = element
	}

	nft.log.Debugw("new cgroup map elements", "value", tmpCGroupMapElement)

	err = conn.SetAddElements(nft.cgroupMap, elements)
	if err != nil {
		return
	}

	tmp := map[int]struct{}{}

	for path := range tmpCGroupMapElement {
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

	for i := len(levels) - 1; i >= 0; i-- {
		err = nft.addCgroupRuleForLevel(conn, levels[i])
		if err != nil {
			return
		}
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	nft.cgroupMapElement = tmpCGroupMapElement

	nft.log.Infow("New cgroup routes added to nft.",
		"size", len(routes),
	)

	nft.dumpNFTableRules()

	return
}

func (nft *NFTManager) RemoveCgroups(paths []string) (err error) {
	defer Wrap(
		&err,
		"remove %d cgroup(s) from nftable",
		len(paths),
	)

	var conn *nftables.Conn
	conn, err = nft.connector.Connect()
	if err != nil {
		return
	}
	elements := []nftables.SetElement{}

	for i := range paths {
		path := nft.removeCgroupRootFromPath(paths[i])

		nft.log.Infow("Removing rule from nft for this cgroup.",
			"cgroup", path,
		)

		if _, ok := nft.cgroupMapElement[path]; !ok {
			nft.log.Debugw("Nothing to do with this cgroup",
				"cgroup", path,
			)
			continue
		}

		elements = append(elements, nft.cgroupMapElement[path])

	}

	err = conn.SetDeleteElements(nft.cgroupMap, elements)
	if err != nil {
		return
	}

	err = conn.Flush()
	if err != nil {
		return
	}

	for i := range paths {
		delete(nft.cgroupMapElement, paths[i])
	}

	nft.dumpNFTableRules()

	return
}

func (nft *NFTManager) AddChainAndRulesForTProxy(tp *config.TProxy) (err error) {
	defer Wrap(
		&err,
		"add chain and rules to nft table for tproxy %#v",
		tp,
	)

	nft.log.Debugw("Adding chain and rules for tproxy.",
		"tproxy", tp,
	)

	var conn *nftables.Conn
	conn, err = nft.connector.Connect()
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

func (nft *NFTManager) Delete() (err error) {
	defer Wrap(&err, "delete NFTManager")

	errs := []error{}

	err = nft.clear()
	if err != nil {
		errs = append(errs, err)
		err = nil
	}

	err = nft.connector.Delete()
	if err != nil {
		errs = append(errs, err)
		err = nil
	}

	if len(errs) != 0 {
		err = errors.Join(errs...)
		return err
	}

	return
}
