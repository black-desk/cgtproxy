package cmd

import (
	"github.com/black-desk/cgtproxy/pkg/cgfsmon"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/cgtproxy/pkg/nftman"
	"github.com/black-desk/cgtproxy/pkg/routeman"
	"github.com/black-desk/cgtproxy/pkg/types"
	"github.com/google/nftables"
	"github.com/google/wire"
	"go.uber.org/zap"
)

type chans struct {
	in  <-chan types.CGroupEvent
	out chan<- types.CGroupEvent
}

func provideChans() chans {
	ch := make(chan types.CGroupEvent)

	return chans{ch, ch}
}

func provideInputChan(chs chans) <-chan types.CGroupEvent {
	return chs.in
}

func provideOutputChan(chs chans) chan<- types.CGroupEvent {
	return chs.out
}

func provideNftConn() (ret *nftables.Conn, err error) {
	var nftConn *nftables.Conn

	nftConn, err = nftables.New(nftables.AsLasting())
	if err != nil {
		return
	}

	ret = nftConn
	return
}

func provideNFTManager(
	root config.CgroupRoot,
	bypass config.Bypass,
	logger *zap.SugaredLogger,
) (
	ret interfaces.NFTManager,
	err error,
) {
	var t *nftman.NFTManager
	t, err = nftman.New(
		nftman.WithCgroupRoot(root),
		nftman.WithBypass(bypass),
		nftman.WithLogger(logger),
	)

	if err != nil {
		return
	}

	ret = t
	return
}

func provideRuleManager(
	t interfaces.NFTManager,
	cfg *config.Config,
	ch <-chan types.CGroupEvent,
	logger *zap.SugaredLogger,
) (
	ret interfaces.RouteManager, err error,
) {
	var r *routeman.RouteManager
	r, err = routeman.New(
		routeman.WithNFTMan(t),
		routeman.WithConfig(cfg),
		routeman.WithCGroupEventChan(ch),
		routeman.WithLogger(logger),
	)

	if err != nil {
		return
	}

	ret = r
	return
}

func provideCgrougMontior(
	cgroupRoot config.CgroupRoot, logger *zap.SugaredLogger,
) (
	interfaces.CGroupMonitor, error,
) {
	return cgfsmon.New(
		cgfsmon.WithCgroupRoot(cgroupRoot),
		cgfsmon.WithLogger(logger),
	)
}

func provideCgroupRoot(cfg *config.Config) config.CgroupRoot {
	return cfg.CgroupRoot
}

func provideBypass(cfg *config.Config) config.Bypass {
	return cfg.Bypass
}

func provideCGTProxy(
	mon interfaces.CGroupMonitor,
	man interfaces.RouteManager,
	logger *zap.SugaredLogger,
	cfg *config.Config,
) (
	interfaces.CGTProxy, error,
) {
	return cgtproxy.New(
		cgtproxy.WithConfig(cfg),
		cgtproxy.WithLogger(logger),
		cgtproxy.WithCGroupMonitor(mon),
		cgtproxy.WithRouteManager(man),
	)
}

var set = wire.NewSet(
	provideBypass,
	provideCgrougMontior,
	provideCgroupRoot,
	provideChans,
	provideCGTProxy,
	provideInputChan,
	provideOutputChan,
	provideRuleManager,
	provideNFTManager,
)
