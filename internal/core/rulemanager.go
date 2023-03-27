package core

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/google/nftables"
)

type ruleManager struct {
	EventChan <-chan *cgroupEvent `inject:"true"`

	netlinkConn *nftables.Conn
}

func (c *Core) newRuleManager() (m *ruleManager, err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
			"Failed to create the nftable rule manager: %w",
			err,
		)
	}()

	m = &ruleManager{}
	if err = c.container.Fill(m); err != nil {
		return
	}

	if m.netlinkConn, err = nftables.New(); err != nil {
		return
	}

	return
}

func (c *Core) runRuleManager() (err error) {
	var m *ruleManager
	if m, err = c.newRuleManager(); err != nil {
		return
	}

	err = m.run()
	return
}

func (m *ruleManager) run() (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
			"Error occurs while running the nftable rules manager: %w",
			err,
		)
	}()

	defer m.removeNftableRules()
	if err = m.initializeNftableRuels(); err != nil {
		return
	}

	for event := range m.EventChan {
		log.Debug().Printf("handle cgroup event %+v", event)
	}
	return
}

func (m *ruleManager) initializeNftableRuels() (err error) {
        m.netlinkConn
	return
}

func (m *ruleManager) removeNftableRules() {
	return
}
