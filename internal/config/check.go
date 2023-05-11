package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/black-desk/deepin-network-proxy-manager/internal/location"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	fstab "github.com/deniswernert/go-fstab"
	"github.com/go-playground/validator/v10"
)

func (c *ConfigV1) check() (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Catch()+
			"Invalid configuration:\n%w",
			err,
		)
	}()

	var validator = validator.New()
	if err = validator.Struct(c); err != nil {
		err = fmt.Errorf("Failed on validation:\n%w", err)
		return
	}

	if c.CgroupRoot == "AUTO" {
		var cgroupRoot string
		if cgroupRoot, err = getCgroupRoot(); err != nil {
			return
		}

		c.CgroupRoot = cgroupRoot
	}

	if c.Rules == nil {
		log.Warning().Printf("No rules in config.")
	}

	if c.Proxies == nil {
		c.Proxies = map[string]*Proxy{}
	}

	if c.TProxies == nil {
		c.TProxies = map[string]*TProxy{}
	}

	rangeExp := regexp.MustCompile(`\[(\d+),(\d+)\)`)

	matchs := rangeExp.FindStringSubmatch(c.Repeater.TProxyPorts)

	if len(matchs) != 3 {
		err = fmt.Errorf(location.Catch()+
			"`tproxy-ports` must be range like [10080,10090), but we get %s",
			c.Repeater.TProxyPorts,
		)
		return
	}

	var (
		begin uint16
		end   uint16

		tmp uint64
	)

	if tmp, err = strconv.ParseUint(matchs[1], 10, 16); err != nil {
		err = fmt.Errorf(location.Catch()+
			"Failed to parse port range begin from %s:\n%w",
			matchs[0], err,
		)
		return
	}
	begin = uint16(tmp)

	if tmp, err = strconv.ParseUint(matchs[2], 10, 16); err != nil {
		err = fmt.Errorf(location.Catch()+
			"Failed to parse port range end from %s:\n%w",
			matchs[1], err,
		)
		return
	}
	end = uint16(tmp)

	if err = c.allocPorts(begin, end); err != nil {
		return
	}

	return
}

func getCgroupRoot() (cgroupRoot string, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Catch()+
			"Failed to get cgroupv2 mount point:\n%w",
			err,
		)
	}()

	var mounts fstab.Mounts
	if mounts, err = fstab.ParseProc(); err != nil {
		return
	}

	var (
		mountFound bool
		fsFile     string
	)
	for i := range mounts {
		mount := mounts[i]
		fsVfsType := mount.VfsType
		fsFile = mount.File

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
