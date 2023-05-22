package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	fstab "github.com/deniswernert/go-fstab"
	"github.com/go-playground/validator/v10"
)

func (c *ConfigV1) check() (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Invalid configuration:\n%w",
			err,
		)
	}()

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
	}

	if c.Rules == nil {
		Log.Warnw("No rules in config.")
	}

	if c.Proxies == nil {
		c.Proxies = map[string]*Proxy{}
	}

	if c.TProxies == nil {
		c.TProxies = map[string]*TProxy{}
	}

	rangeExp := regexp.MustCompile(consts.PortsPattern)

	matchs := rangeExp.FindStringSubmatch(c.Repeater.TProxyPorts)

	if len(matchs) != 3 {
		err = fmt.Errorf(location.Capture()+"%w",
			&ErrWrongPortsPattern{
				Actual: c.Repeater.TProxyPorts,
			},
		)
		return
	}

	var (
		begin uint16
		end   uint16

		tmp uint64
	)

	tmp, err = strconv.ParseUint(matchs[1], 10, 16)
	if err != nil {
		err = fmt.Errorf(location.Capture()+
			"Failed to parse port range begin from %s:\n%w",
			matchs[0], err,
		)
		return
	}
	begin = uint16(tmp)

	tmp, err = strconv.ParseUint(matchs[2], 10, 16)
	if err != nil {
		err = fmt.Errorf(location.Capture()+
			"Failed to parse port range end from %s:\n%w",
			matchs[1], err,
		)
		return
	}
	end = uint16(tmp)

	err = c.allocPorts(begin, end)
	if err != nil {
		return
	}

	return
}

func getCgroupRoot() (cgroupRoot CgroupRoot, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Failed to get cgroupv2 mount point:\n%w",
			err,
		)
	}()

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
		fsFile = CgroupRoot(mount.File)

		if fsVfsType == "cgroup2" {
			mountFound = true
			break
		}
	}

	if !mountFound {
		err = errors.New("`cgroup2` mount point not found in /proc/mounts.")
		return
	}

	cgroupRoot = fsFile

	return
}
