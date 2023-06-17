package config

import (
	"fmt"

	"github.com/black-desk/cgtproxy/internal/consts"
	. "github.com/black-desk/cgtproxy/internal/log"
	. "github.com/black-desk/lib/go/errwrap"
	fstab "github.com/deniswernert/go-fstab"
	"github.com/go-playground/validator/v10"
)

func (c *ConfigV1) check() (err error) {
	defer Wrap(&err, "Invalid configuration.")

	var validator = validator.New()
	err = validator.Struct(c)
	if err != nil {
		err = fmt.Errorf("Failed on validation:\n%w", err)
		return
	}

	if c.CgroupRoot == "AUTO" {
		var cgroupRoot CgroupRoot
		cgroupRoot, err = getCgroupRoot()
		if err != nil {
			return
		}

		c.CgroupRoot = cgroupRoot

		Log.Infow(
			"Cgroup mount point auto detection done.",
			"cgroup root", cgroupRoot,
		)
	}

	if c.Rules == nil {
		Log.Warnw("No rules in config.")
	}

	if c.TProxies == nil {
		c.TProxies = map[string]*TProxy{}
	}

	for name := range c.TProxies {
		tp := c.TProxies[name]
		if tp.Name == "" {
			tp.Name = name
		}
		if tp.DNSHijack != nil && tp.DNSHijack.Addr == nil {
			addr := consts.IPv4LocalhostStr
			tp.DNSHijack.Addr = &addr
		}
	}

	return
}

func getCgroupRoot() (cgroupRoot CgroupRoot, err error) {
	defer Wrap(&err, "Failed to get cgroupv2 mount point.")

	var mounts fstab.Mounts
	mounts, err = fstab.ParseProc()
	if err != nil {
		return
	}

	var (
		mountFound bool
		fsFile     CgroupRoot
	)
	for i := range mounts {
		mount := mounts[i]
		fsVfsType := mount.VfsType

		if fsVfsType != "cgroup2" {
			continue
		}

		fsFile = CgroupRoot(mount.File)
		mountFound = true
	}

	if !mountFound {
		err = ErrCannotFoundCgroupv2Mount
		return
	}

	cgroupRoot = fsFile

	return
}
