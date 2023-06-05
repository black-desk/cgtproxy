package config

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"

	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
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

	{
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
	}

	{
		if c.RouteTable == 0 {
			c.RouteTable = rand.Int()
		}
	}

	{
		if c.Mark == 0 {
			c.Mark = RerouteMark(rand.Int())
		}
	}

	{
		if c.Rules == nil {
			Log.Warnw("No rules in config.")
		}
	}

	{
		if c.Proxies == nil {
			c.Proxies = map[string]*Proxy{}
		}

		if c.TProxies == nil {
			c.TProxies = map[string]*TProxy{}
		}

		for name := range c.TProxies {
			tp := c.TProxies[name]
			if tp.Name == "" {
				tp.Name = name
			}
		}
	}

	{
		if c.Repeater == nil {
			return
		}

		rangeExp := regexp.MustCompile(consts.PortsPattern)

		matchs := rangeExp.FindStringSubmatch(c.Repeater.TProxyPorts)

		if len(matchs) != 3 {
			err = &ErrWrongPortsPattern{
				Actual: c.Repeater.TProxyPorts,
			}
			Wrap(&err)

			return
		}

		var (
			begin uint16
			end   uint16

			tmp uint64
		)

		tmp, err = strconv.ParseUint(matchs[1], 10, 16)
		if err != nil {
			Wrap(&err,
				"Failed to parse port range begin from %s.",
				matchs[0],
			)
			return
		}
		begin = uint16(tmp)

		tmp, err = strconv.ParseUint(matchs[2], 10, 16)
		if err != nil {
			Wrap(&err,
				"Failed to parse port range end from %s.",
				matchs[1],
			)
			return
		}
		end = uint16(tmp)

		err = c.allocPorts(begin, end)
		if err != nil {
			return
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
		err = errors.New("`cgroup2` mount point not found in /proc/mounts.")
		return
	}

	cgroupRoot = fsFile

	return
}
