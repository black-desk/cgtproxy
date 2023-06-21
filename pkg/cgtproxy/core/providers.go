package core

import (
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/core/internal/monitor"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/core/internal/rulemanager"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/core/internal/table"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/core/internal/watcher"
	"github.com/google/nftables"
	"github.com/google/wire"
)

func provideWatcher(cgroupRoot config.CgroupRoot,
) (
	ret *watcher.Watcher, err error,
) {
	var w *watcher.Watcher

	w, err = watcher.New(watcher.WithCgroupRoot(cgroupRoot))
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
) (
	ret *table.Table,
	err error,
) {
	var t *table.Table
	t, err = table.New(
		table.WithCgroupRoot(root),
		table.WithBypass(bypass),
	)

	if err != nil {
		return
	}

	ret = t
	return

}

func provideRuleManager(
	t *table.Table, cfg *config.Config, ch <-chan *types.CgroupEvent,
) (
	ret *rulemanager.RuleManager, err error,
) {
	var r *rulemanager.RuleManager
	r, err = rulemanager.New(
		rulemanager.WithTable(t),
		rulemanager.WithConfig(cfg),
		rulemanager.WithCgroupEventChan(ch),
	)

	if err != nil {
		return
	}

	ret = r
	return
}

func provideMonitor(
	ch chan<- *types.CgroupEvent,
	w *watcher.Watcher,
	root config.CgroupRoot,
) (
	ret *monitor.Monitor, err error,
) {
	var m *monitor.Monitor

	m, err = monitor.New(
		monitor.WithOutput(ch),
		monitor.WithWatcher(w),
		monitor.WithCgroupRoot(root),
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
	w *watcher.Watcher, m *monitor.Monitor, r *rulemanager.RuleManager,
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
