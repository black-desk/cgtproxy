// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

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

	for name := range c.TProxies {
		tp := c.TProxies[name]
		if tp.Name == "" {
			tp.Name = name
		}
		if tp.DNSHijack != nil && tp.DNSHijack.IP == nil {
			addr := IPv4LocalhostStr
			tp.DNSHijack.IP = &addr
		}
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
