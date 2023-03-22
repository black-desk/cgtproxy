package core

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func (c *Core) Run() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
                        "Error occurs while running the core: %w",
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
