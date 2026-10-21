// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import "go.uber.org/zap"

type Config struct {
	Version string `yaml:"version" validate:"required,eq=1"`

	CgroupRoot CGroupRoot `yaml:"cgroup-root" validate:"required,dirpath|eq=AUTO"`
	// Gateway applies the root cgroup target to forwarded traffic.
	// Forwarded traffic is left unchanged unless this is enabled.
	Gateway bool `yaml:"gateway"`
	// Bypass describes the bypass rules apply to all the TPROXY servers.
	// If the destination matched in Bypass, the traffic will not be touched.
	Bypass   Bypass             `yaml:"bypass" validate:"dive,ipv4|cidrv4|ipv6|cidrv6"`
	TProxies map[string]*TProxy `yaml:"tproxies" validate:"dive"`
	Rules    []Rule             `yaml:"rules" validate:"dive"`
	// RouteTable identifies a dedicated table for TPROXY local default routes.
	// Valid IDs are 1-4294967295, excluding 253 (default), 254 (main), and 255 (local).
	// cgtproxy removes the routes it installs when it stops.
	RouteTable int `yaml:"route-table" validate:"gt=0,lte=4294967295,ne=253,ne=254,ne=255"`

	log *zap.SugaredLogger `yaml:"-"`
	raw []byte
}

type Bypass []string

type CGroupRoot string

// Rule describes a rule about how to handle traffic comes from a cgroup.
type Rule struct {
	// Match is an regex expression
	// to match an cgroup path relative to the root of cgroupfs.
	Match string `yaml:"match" validate:"required"`

	// TProxy means that the traffic comes from this cgroup
	// should be redirected to a TPROXY server.
	TProxy string `yaml:"tproxy" validate:"required_without_all=Drop Direct,excluded_with=Drop Direct"`
	// Drop means that the traffic comes from this cgroup will be dropped.
	Drop bool `yaml:"drop" validate:"required_without_all=TProxy Direct,excluded_with=TProxy Direct"`
	// Direct means that the traffic comes from this cgroup will not be touched.
	Direct bool `yaml:"direct" validate:"required_without_all=TProxy Drop,excluded_with=TProxy Drop"`
}

// TProxy describes a TPROXY server.
type TProxy struct {
	// Name is the nonempty tproxies map key. It generates nftables chains named
	// Name and Name+"-MARK", plus Name+"-DNS" when DNSHijack is configured.
	// Each generated name must be unique across proxies, differ from the internal
	// chain names in internal/types/chains.go, contain no NUL, and fit in 255 bytes.
	Name   string `yaml:"-"`
	NoUDP  bool   `yaml:"no-udp"`
	NoIPv6 bool   `yaml:"no-ipv6"`
	Port   uint16 `yaml:"port" validate:"required"`
	// Mark is the fire wall mark used to identify the TPROXY server
	// and trigger reroute operation of netfliter
	// from OUTPUT to PREROUTING internally.
	// It **NOT** means that this TPROXY server
	// must send traffic with the fwmark.
	// This mark is designed to be changeable for user
	// to make sure this mark is not conflict
	// with any fire wall mark in use.
	Mark FireWallMark `yaml:"mark" validate:"required"`
	// DNSHijack will hijack the dns request traffic
	// should redirect to this TPROXY server,
	// and send them to directory to a dns server described in DNSHijack.
	// This option is for fake-ip.
	DNSHijack *DNSHijack `yaml:"dns-hijack"`
}

type FireWallMark uint32

type DNSHijack struct {
	IP *string `yaml:"ip" validate:"omitempty,ip4_addr"`
	// Port is required when DNS hijacking is configured; valid range: 1-65535.
	// Set 53 explicitly for a resolver listening on the standard DNS port.
	Port uint16 `yaml:"port" validate:"required"`
	// If TCP is set to true,
	// tcp traffic will be hijacked, too,
	// when it's destination port is 53.
	TCP bool `yaml:"tcp"`
}
