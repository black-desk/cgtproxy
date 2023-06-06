package config

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

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

			if strings.HasSuffix(tp.Name, "-MARK") {
				err = &ErrBadProxyName{
					Actual: tp.Name,
				}
				Wrap(&err)
				return
			}
		}
	}

	if c.Repeater != nil {
		var (
			begin uint64
			end   uint64
		)

		begin, end, err = parseRange(c.Repeater.TProxyPorts)
		if err != nil {
			return
		}

		err = c.allocPorts(uint16(begin), uint16(end))
		if err != nil {
			return
		}
	}

	{
		var (
			begin uint64
			end   uint64
		)

		begin, end, err = parseRange(c.Marks)
		if err != nil {
			return
		}

		err = c.allocMarks(int(begin), int(end))
		if err != nil {
			return
		}
	}

	return
}

func parseRange(str string) (begin uint64, end uint64, err error) {
	defer Wrap(&err, "Failed to parse range.")

	rangeExp := regexp.MustCompile(consts.PortsPattern)

	matchs := rangeExp.FindStringSubmatch(str)

	if len(matchs) != 3 {
		err = &ErrBadRange{
			Actual: str,
		}
		Wrap(&err)

		return
	}

	begin, err = strconv.ParseUint(matchs[1], 10, 16)
	if err != nil {
		Wrap(&err,
			"Failed to parse range begin from %s.",
			matchs[0],
		)
		return
	}

	end, err = strconv.ParseUint(matchs[2], 10, 16)
	if err != nil {
		Wrap(&err,
			"Failed to parse range end from %s.",
			matchs[1],
		)
		return
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
