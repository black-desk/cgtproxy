// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/black-desk/cgtproxy/internal/types"
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

	marks := make(map[FireWallMark]string, len(c.TProxies))
	chains := map[string]string{
		types.ChainCgroupRoot:    "internal chain",
		types.ChainOutputMangle:  "internal chain",
		types.ChainOutputNAT:     "internal chain",
		types.ChainPrerouting:    "internal chain",
		types.ChainForwardMangle: "internal chain",
		types.ChainPreroutingNAT: "internal chain",
	}
	for name, tp := range c.TProxies {
		if tp == nil {
			return fmt.Errorf("tproxies[%q]: must not be null", name)
		}
		if previous, exists := marks[tp.Mark]; exists {
			return fmt.Errorf("tproxies[%q].mark: %d is also used by tproxies[%q].mark", name, tp.Mark, previous)
		}
		marks[tp.Mark] = name
		if tp.Name == "" {
			tp.Name = name
		}
		if tp.Name == "" {
			return fmt.Errorf("tproxies[%q]: proxy name must not be empty", name)
		}
		proxyChains := []string{tp.Name, tp.Name + "-MARK"}
		if tp.DNSHijack != nil {
			proxyChains = append(proxyChains, tp.Name+"-DNS")
		}
		for _, chain := range proxyChains {
			// NFT_NAME_MAXLEN includes the terminating NUL byte.
			if len(chain) > 255 || strings.ContainsRune(chain, '\x00') {
				return fmt.Errorf("tproxies[%q]: generated chain name must be at most 255 bytes without NUL", name)
			}
			if owner, exists := chains[chain]; exists {
				return fmt.Errorf("tproxies[%q]: generated chain %q conflicts with %s", name, chain, owner)
			}
			chains[chain] = fmt.Sprintf("tproxies[%q]", name)
		}
	}

	for i, rule := range c.Rules {
		if _, err := regexp.Compile(rule.Match); err != nil {
			return fmt.Errorf("rules[%d].match: %w", i, err)
		}
		if rule.TProxy != "" {
			if _, exists := c.TProxies[rule.TProxy]; !exists {
				return fmt.Errorf("rules[%d].tproxy: undefined proxy %q", i, rule.TProxy)
			}
		}
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

	// Every component must use the same root when deriving cgroup paths and
	// socket ancestry levels, including roots containing "." or repeated "/".
	c.CgroupRoot = CGroupRoot(filepath.Clean(string(c.CgroupRoot)))

	if c.Rules == nil {
		c.log.Warnw("No rules in config.")
	}

	if c.TProxies == nil {
		c.TProxies = map[string]*TProxy{}
	}

	for name := range c.TProxies {
		tp := c.TProxies[name]
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
