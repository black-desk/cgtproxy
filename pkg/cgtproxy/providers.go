package cgtproxy

import (
	"github.com/black-desk/cgtproxy/internal/nftman"
	"github.com/black-desk/cgtproxy/internal/routeman"
	"github.com/black-desk/cgtproxy/pkg/cgfsmon"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/cgtproxy/pkg/types"
	"github.com/google/nftables"
	"go.uber.org/zap"
)

type chans struct {
	in  <-chan types.CgroupEvent
	out chan<- types.CgroupEvent
}

func provideChans() chans {
	ch := make(chan types.CgroupEvent)

	return chans{ch, ch}
}

func provideInputChan(chs chans) <-chan types.CgroupEvent {
	return chs.in
}

func provideOutputChan(chs chans) chan<- types.CgroupEvent {
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

func provideTable(
	root config.CgroupRoot,
	bypass config.Bypass,
	logger *zap.SugaredLogger,
) (
	ret *nftman.Table,
	err error,
) {
	var t *nftman.Table
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
	t *nftman.Table,
	cfg *config.Config,
	ch <-chan types.CgroupEvent,
	logger *zap.SugaredLogger,
) (
	ret *routeman.RouteManager, err error,
) {
	var r *routeman.RouteManager
	r, err = routeman.New(
		routeman.WithTable(t),
		routeman.WithConfig(cfg),
		routeman.WithCgroupEventChan(ch),
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

func provideComponents(
	m interfaces.CGroupMonitor,
	r *routeman.RouteManager,
) *components {
	return &components{m: m, r: r}
}

