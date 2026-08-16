// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import "go.uber.org/zap"

type Config struct {
	Version string `yaml:"version" validate:"required,eq=1"`

	CgroupRoot CGroupRoot `yaml:"cgroup-root" validate:"required,dirpath|eq=AUTO"`
	// Bypass describes the bypass rules apply to all the TPROXY servers.
	// If the destination matched in Bypass, the traffic will not be touched.
	Bypass   Bypass              `yaml:"bypass" validate:"dive,ipv4|cidrv4|ipv6|cidrv6"`
	TProxies map[string]*TProxy  `yaml:"tproxies" validate:"dive"`
	Gateways map[string]*Gateway `yaml:"gateways" validate:"dive"`
	Rules    []Rule              `yaml:"rules" validate:"dive"`
	// The route table number cgtproxy will create to route TPROXY traffic.
	// This table will be removed when cgtproxy stopped.
	RouteTable int `yaml:"route-table" validate:"required"`

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
	TProxy string `yaml:"tproxy" validate:"required_without_all=Drop Direct Gateway,excluded_with=Drop Direct Gateway"`
	// Drop means that the traffic comes from this cgroup will be dropped.
	Drop bool `yaml:"drop" validate:"required_without_all=TProxy Direct Gateway,excluded_with=TProxy Direct Gateway"`
	// Direct means that the traffic comes from this cgroup will not be touched.
	Direct bool `yaml:"direct" validate:"required_without_all=TProxy Drop Gateway,excluded_with=TProxy Drop Gateway"`
	// Gateway means that the traffic comes from this cgroup
	// should be routed to a specific next-hop gateway,
	// which is expected to do transparent proxy by itself.
	Gateway string `yaml:"gateway" validate:"required_without_all=TProxy Drop Direct,excluded_with=TProxy Drop Direct"`
}

// TProxy describes a TPROXY server.
type TProxy struct {
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

// Gateway describes a next-hop gateway which does transparent proxy.
// Traffic routed to a Gateway target will be marked and sent
// to this gateway through a dedicated route table,
// so the gateway can intercept it on its own.
type Gateway struct {
	Name string `yaml:"-"`
	// IP is the address of the next-hop gateway.
	IP string `yaml:"ip" validate:"required,ip"`
	// Dev is the network interface used to reach the gateway.
	// It must be specified so that the gateway is treated as on-link
	// and no recursive route lookup is needed.
	Dev string `yaml:"dev" validate:"required"`
	// Mark is the fire wall mark used to trigger the reroute operation
	// of netfilter from OUTPUT to the dedicated route table.
	// It should not conflict with any fire wall mark in use,
	// including marks used by TProxy servers.
	Mark FireWallMark `yaml:"mark" validate:"required"`
}

type FireWallMark uint32

type DNSHijack struct {
	IP   *string `yaml:"ip" validate:"ip4_addr"`
	Port uint16  `yaml:"port"`
	// If TCP is set to true,
	// tcp traffic will be hijacked, too,
	// when it's destination port is 53.
	TCP bool `yaml:"tcp"`
}
