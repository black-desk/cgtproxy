package config

import (
	. "github.com/black-desk/cgtproxy/internal/log"
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

	for _, tp := range c.TProxies {

		if tp.Port != 0 {
			continue
		}

		if begin >= end {
			err = ErrTooFewPorts
			return
		}

		tp.Port = begin
		Log.Debugw("Allocate port for tproxy.",
			"tproxy", tp,
		)

		begin++
	}

	return
}

func (c *ConfigV1) allocMarks(begin, end int) (err error) {
	for _, tp := range c.TProxies {
		if tp.Mark != 0 {
			continue
		}

		if begin >= end {
			err = ErrTooFewMarks
			return
		}

		tp.Mark = RerouteMark(begin)

		Log.Debugw("Allocate mark for tproxy.",
			"tproxy", tp,
		)

		begin++

	}
	return
}
