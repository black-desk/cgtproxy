package core

import (
	"sync"

	"github.com/black-desk/cgtproxy/internal/config"
	"github.com/black-desk/cgtproxy/internal/core/monitor"
	"github.com/black-desk/cgtproxy/internal/core/repeater"
	"github.com/black-desk/cgtproxy/internal/core/rulemanager"
	"github.com/black-desk/cgtproxy/internal/core/table"
	"github.com/black-desk/cgtproxy/internal/core/watcher"
	"github.com/black-desk/cgtproxy/internal/types"
	"github.com/google/nftables"

	"github.com/google/wire"
)

func provideConfig(c *Core) (cfg *config.Config, err error) {
	if c.cfg == nil {
		err = ErrConfigMissing
		return
	}

	cfg = c.cfg
	return
}

var (
	_watcherOnce sync.Once
	_watcher     *watcher.Watcher
	_wathcerErr  error
)

func provideWatcher(cgroupRoot config.CgroupRoot,
) (
	ret *watcher.Watcher, err error,
) {
	_watcherOnce.Do(func() {
		_watcher, _wathcerErr = watcher.New(
			watcher.WithCgroupRoot(cgroupRoot),
		)
	})

	if _wathcerErr != nil {
		err = _wathcerErr
		return
	}

	ret = _watcher
	return

}

var (
	_chanOnce sync.Once
	_ch       chan *types.CgroupEvent
)

func provideChan() (ret chan *types.CgroupEvent) {
	_chanOnce.Do(func() {
		_ch = make(chan *types.CgroupEvent)
	})

	ret = _ch
	return
}

func provideInputChan() (ret <-chan *types.CgroupEvent) {
	return provideChan()
}

func provideOutputChan() (ret chan<- *types.CgroupEvent) {
	return provideChan()
}

var (
	_nftConnOnce sync.Once
	_nftConn     *nftables.Conn
	_nftErr      error
)

func provideNftConn() (ret *nftables.Conn, err error) {
	_nftConnOnce.Do(func() {
		_nftConn, _nftErr = nftables.New(nftables.AsLasting())
	})

	if _nftErr != nil {
		err = _nftErr
		return
	}

	ret = _nftConn
	return
}

var (
	_tableOnce sync.Once
	_table     *table.Table
	_tableErr  error
)

func provideTable(
	root config.CgroupRoot,
	bypass *config.Bypass,
) (
	ret *table.Table,
	err error,
) {
	_tableOnce.Do(func() {
		_table, _tableErr = table.New(
			table.WithCgroupRoot(root),
			table.WithBypass(bypass),
		)
	})

	if _tableErr != nil {
		err = _tableErr
		return
	}

	ret = _table
	return

}

var (
	_ruleManagerOnce sync.Once
	_ruleMananger    *rulemanager.RuleManager
	_ruleManagerErr  error
)

func provideRuleManager(
	t *table.Table, cfg *config.Config, ch <-chan *types.CgroupEvent,
) (
	ret *rulemanager.RuleManager, err error,
) {
	_ruleManagerOnce.Do(func() {
		_ruleMananger, _ruleManagerErr = rulemanager.New(
			rulemanager.WithTable(t),
			rulemanager.WithConfig(cfg),
			rulemanager.WithCgroupEventChan(ch),
		)
	})

	if _ruleManagerErr != nil {
		err = _ruleManagerErr
		return
	}

	ret = _ruleMananger
	return
}

func provideRepeater() (*repeater.Repeater, error) {
	return repeater.New()
}

var (
	_monitorOnce sync.Once
	_monitor     *monitor.Monitor
	_monitorErr  error
)

func provideMonitor(
	ch chan<- *types.CgroupEvent,
	w *watcher.Watcher,
	root config.CgroupRoot,
) (
	ret *monitor.Monitor, err error,
) {
	_monitorOnce.Do(func() {
		_monitor, _monitorErr = monitor.New(
			monitor.WithOutput(ch),
			monitor.WithWatcher(w),
			monitor.WithCgroupRoot(root),
		)
	})
	if _monitorErr != nil {
		err = _monitorErr
		return
	}

	ret = _monitor
	return
}

func provideCgroupRoot(cfg *config.Config) config.CgroupRoot {
	return cfg.CgroupRoot
}

func provideBypass(cfg *config.Config) *config.Bypass {
	return cfg.Bypass
}

var set = wire.NewSet(
	provideConfig,
	provideWatcher,
	provideInputChan,
	provideOutputChan,
	provideTable,
	provideRuleManager,
	provideRepeater,
	provideMonitor,
	provideCgroupRoot,
	provideBypass,
)
