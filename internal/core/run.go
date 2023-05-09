package core

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/black-desk/deepin-network-proxy-manager/internal/core/monitor"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/repeater"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/rulemanager"
	"github.com/black-desk/deepin-network-proxy-manager/internal/location"
)

func (c *Core) Run() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(location.Catch()+
			"Error occurs while running the core:\n%w",
			err,
		)
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	if err = c.start(); err != nil {
		return err
	}

	go func() {
		<-sigChan
		c.cancel()
	}()

	return c.pool.Wait()
}

func (c *Core) start() (err error) {
	c.pool.Go(c.runMonitor)
	c.pool.Go(c.runRepeater)
	c.pool.Go(c.runRuleManager)
	return
}

func (c *Core) runMonitor() (err error) {
	var m *monitor.Monitor
	if m, err = monitor.New(c.container); err != nil {
		return
	}

	err = m.Run()
	return
}

func (c *Core) runRuleManager() (err error) {
	var r *rulemanager.RuleManager
	if r, err = rulemanager.New(c.container); err != nil {
		return
	}

	err = r.Run()
	return
}

func (c *Core) runRepeater() (err error) {
	var r *repeater.Repeater
	if r, err = repeater.New(c.container); err != nil {
		return
	}

	err = r.Run()
	return
}
