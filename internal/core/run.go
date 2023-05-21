package core

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/black-desk/deepin-network-proxy-manager/internal/core/monitor"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/repeater"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/rulemanager"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/watcher"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
)

func (c *Core) Run() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Capture()+
			"Error occurs while running the core:\n%w",
			err,
		)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	c.start()

	go func() {
		select {
		case <-c.ctx.Done():
		case <-sigChan:
		}
		c.cancel()
	}()

	return c.pool.Wait()
}

func (c *Core) start() {
	c.pool.Go(c.runWatcher)
	c.pool.Go(c.runMonitor)
	c.pool.Go(c.runRuleManager)

	if c.cfg.Repeater == nil {
		return
	}

	c.pool.Go(c.runRepeater)

	return
}

func (c *Core) runMonitor() (err error) {
	defer c.cancel()

	var m *monitor.Monitor
	m, err = monitor.New() // FIXME(black_desk): add opts later
	if err != nil {
		return
	}

	err = m.Run()
	return
}

func (c *Core) runRuleManager() (err error) {
	defer c.cancel()

	var r *rulemanager.RuleManager
	r, err = rulemanager.New() // FIXME(black_desk): add opts later
	if err != nil {
		return
	}

	err = r.Run()
	return
}

func (c *Core) runRepeater() (err error) {
	defer c.cancel()

	var r *repeater.Repeater
	r, err = repeater.New() // FIXME(black_desk): add opts later

	if err != nil {
		return
	}

	err = r.Run()
	return
}

func (c *Core) runWatcher() (err error) {
	defer c.cancel()

	var w *watcher.Watcher
	w, err = watcher.New()

	if err != nil {
		return
	}

	err = w.Run()
	return
}
