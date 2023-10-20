package cgtproxy

import (
	"github.com/black-desk/cgtproxy/internal/cgmon"
	"github.com/black-desk/cgtproxy/internal/fswatcher"
	"github.com/black-desk/cgtproxy/internal/nftman"
	"github.com/black-desk/cgtproxy/internal/routeman"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/cgtproxy/pkg/types"
	"github.com/google/nftables"
	"github.com/google/wire"
	"go.uber.org/zap"
)

func provideWatcher(
	cgroupRoot config.CgroupRoot,
	logger *zap.SugaredLogger,
) (
	ret *fswatcher.Watcher, err error,
) {
	var w *fswatcher.Watcher

	w, err = fswatcher.New(
		fswatcher.WithCgroupRoot(cgroupRoot),
		fswatcher.WithLogger(logger),
	)
	if err != nil {
		return
	}

	ret = w
	return
}

type chans struct {
	in  <-chan *types.CgroupEvent
	out chan<- *types.CgroupEvent
}

func provideChans() chans {
	ch := make(chan *types.CgroupEvent)

	return chans{ch, ch}
}

func provideInputChan(chs chans) <-chan *types.CgroupEvent {
	return chs.in
}

func provideOutputChan(chs chans) chan<- *types.CgroupEvent {
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
	ch <-chan *types.CgroupEvent,
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

func provideMonitor(
	ch chan<- *types.CgroupEvent,
	w *fswatcher.Watcher,
	root config.CgroupRoot,
	logger *zap.SugaredLogger,
) (
	ret interfaces.CgroupMonitor, err error,
) {
	var m *cgmon.FSMonitor

	m, err = cgmon.New(
		cgmon.WithOutput(ch),
		cgmon.WithWatcher(w),
		cgmon.WithCgroupRoot(root),
		cgmon.WithLogger(logger),
	)
	if err != nil {
		return
	}

	ret = m
	return
}

func provideCgroupRoot(cfg *config.Config) config.CgroupRoot {
	return cfg.CgroupRoot
}

func provideBypass(cfg *config.Config) config.Bypass {
	return cfg.Bypass
}

func provideComponents(
	w *fswatcher.Watcher, m interfaces.CgroupMonitor, r *routeman.RouteManager,
) *components {
	return &components{w: w, m: m, r: r}
}

var set = wire.NewSet(
	provideComponents,
	provideWatcher,
	provideChans,
	provideInputChan,
	provideOutputChan,
	provideTable,
	provideRuleManager,
	provideMonitor,
	provideCgroupRoot,
	provideBypass,
)
