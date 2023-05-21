package core

import (
	"context"
	"sync"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/monitor"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/repeater"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/rulemanager"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/table"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/watcher"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
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

func provideWatcher(
	ctx context.Context, cgroupRoot config.CgroupRoot,
) (
	ret *watcher.Watcher, err error,
) {
	_watcherOnce.Do(func() {
		_watcher, _wathcerErr = watcher.New(
			watcher.WithContext(ctx),
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

var (
	_nftConnOnce sync.Once
	_nftConn     *nftables.Conn
	_nftErr      error
)

func provideInputChan() (ret <-chan *types.CgroupEvent) {
	return provideChan()
}

func provideOutputChan() (ret chan<- *types.CgroupEvent) {
	return provideChan()
}

func provideNftConn() (ret *nftables.Conn, err error) {
	_nftConnOnce.Do(func() {
		_nftConn, _nftErr = nftables.New()
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
	conn *nftables.Conn,
	mark config.RerouteMark,
	root config.CgroupRoot,
) (
	ret *table.Table,
	err error,
) {
	_tableOnce.Do(func() {
		_table, _tableErr = table.New(
			table.WithConn(conn),
			table.WithRerouteMark(mark),
			table.WithCgroupRoot(root),
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
	ctx context.Context, ch chan<- *types.CgroupEvent, w *watcher.Watcher,
) (
	ret *monitor.Monitor, err error,
) {
	_monitorOnce.Do(func() {
		_monitor, _monitorErr = monitor.New(
			monitor.WithCtx(ctx),
			monitor.WithOutput(ch),
			monitor.WithWatcher(w),
		)
	})
	if _monitorErr != nil {
		err = _monitorErr
		return
	}

	ret = _monitor
	return
}

func provideContext(c *Core) context.Context {
	return c.ctx
}

func provideCgroupRoot(cfg *config.Config) config.CgroupRoot {
	return cfg.CgroupRoot
}

func provideRerouteMark(cfg *config.Config) config.RerouteMark {
	return cfg.Mark
}

var set = wire.NewSet(
	provideConfig,
	provideWatcher,
	provideInputChan,
	provideOutputChan,
	provideNftConn,
	provideTable,
	provideRuleManager,
	provideRepeater,
	provideMonitor,
	provideContext,
	provideCgroupRoot,
	provideRerouteMark,
)
