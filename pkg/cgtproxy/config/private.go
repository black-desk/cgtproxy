package config

import (
	"fmt"
	"os"

	. "github.com/black-desk/lib/go/errwrap"
	"github.com/go-playground/validator/v10"
)

func (c *Config) check() (err error) {
	defer Wrap(&err, "check configuration")

	var validator = validator.New()
	err = validator.Struct(c)
	if err != nil {
		err = fmt.Errorf("validator: %w", err)
		return
	}

	if c.CgroupRoot == "AUTO" {
		var cgroupRoot CGroupRoot
		cgroupRoot, err = getCgroupRoot()
		if err != nil {
			return
		}

		c.CgroupRoot = cgroupRoot

		c.log.Infow(
			"Cgroup mount point auto detection done.",
			"cgroup root", cgroupRoot,
		)
	}

	if c.Rules == nil {
		c.log.Warnw("No rules in config.")
	}

	if c.TProxies == nil {
		c.TProxies = map[string]*TProxy{}
	}

	var gatewayTproxy string

	for name := range c.TProxies {
		tp := c.TProxies[name]
		if tp.Name == "" {
			tp.Name = name
		}

		if tp.DNSHijack != nil && tp.DNSHijack.IP == nil {
			addr := IPv4LocalhostStr
			tp.DNSHijack.IP = &addr
		}

		if !tp.Gateway {
			continue
		}

		if gatewayTproxy == "" {
			gatewayTproxy = tp.Name
			continue
		}

		err = fmt.Errorf("Multiple gateway targets found: %s and %s",
			gatewayTproxy, tp.Name)
		return

	}

	if gatewayTproxy != "" {
		c.log.Debugw("Gateway mode enabled", "tproxy", gatewayTproxy)
	}

	return
}

func getCgroupRoot() (cgroupRoot CGroupRoot, err error) {
	defer Wrap(&err, "get cgroupv2 mount point")

	paths := []string{"/sys/fs/cgroup/unified", "/sys/fs/cgroup"}

	for i := range paths {
		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(paths[i])
		if err != nil {
			continue
		}

		if !fileInfo.IsDir() {
			continue
		}

		cgroupRoot = CGroupRoot(paths[i])
		return
	}

	err = ErrCannotFoundCgroupv2Root
	return
}
