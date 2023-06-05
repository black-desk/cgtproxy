package config

import (
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
)

func (c *ConfigV1) allocPorts(begin, end uint16) (err error) {
	defer Wrap(&err, "Failed to allocate mark for proxy.")

	for name := range c.Proxies {
		p := c.Proxies[name]

		if p.TProxy != nil {
			panic("this should never happened")
		}

		p.TProxy = &TProxy{
			Name:   "repeater-" + name,
			NoUDP:  !p.UDP,
			NoIPv6: p.NoIPv6,
			Addr:   &c.Repeater.Listens[0],
			Port:   0, // NOTE(black_desk): alloc later
		}

		c.TProxies["repeater-"+name] = p.TProxy

		Log.Debugw("Create tproxy for proxy.",
			"tproxy", p.TProxy.Name, "proxy", name,
		)
	}

	for name := range c.TProxies {
		tp := c.TProxies[name]

		if tp.Port != 0 {
			continue
		}

		if begin >= end {
			err = ErrTooFewPorts
			return
		}

		tp.Port = begin
		Log.Debugw("Allocate port for tproxy",
			"port", tp.Port, "tproxy", tp,
		)

		begin++
	}

	return
}
