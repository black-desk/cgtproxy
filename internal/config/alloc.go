package config

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
)

func (c *ConfigV1) allocPorts(begin, end uint16) (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Failed to allocate mark for proxy: %w",
			err,
		)
	}()

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

		log.Debug().Printf("Create tproxy %s for proxy %s",
			p.TProxy.Name, name)
	}

	for name := range c.TProxies {
		tp := c.TProxies[name]

		if tp.Name == "" {
			tp.Name = name
		}

		if tp.Port != 0 {
			continue
		}

		if begin >= end {
			err = fmt.Errorf(location.Capture()+
				"%w %#v", ErrTooFewPorts, tp,
			)
			return
		}

		tp.Port = begin
		log.Debug().Printf("Allocate port %d for tproxy %s",
			tp.Port, tp.String(),
		)

		begin++
	}

	return
}
