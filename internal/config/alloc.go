package config

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/location"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/google/uuid"
)

func (c *ConfigV1) allocPorts(begin, end uint16) (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Catch()+
			"Failed to allocate mark for proxy: %w",
			err,
		)
	}()

	c.collect()

	for tp := range c.TProxies {
		if tp.Port != 0 {
			continue
		}

		if begin >= end {
			err = fmt.Errorf(location.Catch()+
				"%w %#v", ErrTooFewPorts, tp,
			)
			return
		}

		tp.Port = begin
		log.Debug().Printf("Allocate port %d for tproxy %s",
			tp.Port, tp.Name,
		)

		begin++
	}

	return
}

func (c *ConfigV1) collect() {
	for i := range c.Rules {
		c.collectProxies(c.Rules[i])
		c.collectTProxies(c.Rules[i])
	}
	return
}

func (c *ConfigV1) collectProxies(rule Rule) {
	if rule.Proxy == nil {
		return
	}

	if _, ok := c.Proxies[rule.Proxy]; ok {
		return
	}

	if rule.Proxy.Name == "" {
		rule.Proxy.Name = uuid.NewString()
	}

	rule.Proxy.TProxy = &TProxy{
		Name:   "repeater-" + rule.Proxy.Name,
		NoUDP:  !rule.Proxy.UDP,
		NoIPv6: rule.Proxy.NoIPv6,
		Addr:   &c.Repeater.Listens[0],
		Port:   0, // NOTE(black_desk): alloc later
	}

	log.Debug().Printf("Create tproxy for proxy:\n%v", rule.Proxy)

	c.TProxies[rule.Proxy.TProxy] = struct{}{}
	c.Proxies[rule.Proxy] = struct{}{}

	return
}

func (c *ConfigV1) collectTProxies(rule Rule) {
	if rule.TProxy == nil {
		return
	}

	if _, ok := c.TProxies[rule.TProxy]; ok {
		return
	}

	c.TProxies[rule.TProxy] = struct{}{}

	return
}
