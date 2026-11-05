// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import "go.uber.org/zap"

// Config describes the YAML configuration for desktop transparent proxying.
// Unknown fields, including the removed drop target, are rejected. Each rule
// selects either direct: true or a named tproxy.
//
// cgtproxy provides proxy convenience, without network isolation or a guarantee
// that traffic always uses a proxy. Unexpected runtime errors are reported and
// cause cgtproxy to remove its rules and routes and exit. Cleanup failures are
// logged with the state that could not be removed.
//
// See the [example configuration] for a complete setup.
//
// [example configuration]: https://github.com/black-desk/cgtproxy/blob/master/misc/config/example.yaml
type Config struct {
	// Version is the configuration format version; currently only 1 is supported.
	Version string `yaml:"version" validate:"required,eq=1"`

	// CgroupRoot is the cgroup v2 mount path, or AUTO to discover it.
	CgroupRoot CGroupRoot `yaml:"cgroup-root" validate:"required,dirpath|eq=AUTO"`
	// Gateway selects a target for traffic routed through this host, such as
	// LAN, virtual machine, or Docker bridge traffic. Omit it to leave forwarded
	// traffic unchanged, use direct: true to explicitly leave it unproxied, or
	// select a named tproxy. It is independent of local cgroup Rules, including /.
	// All forwarding sources share this target because their process cgroups
	// cannot be determined on this host.
	//
	// In PREROUTING, connection replies, local destinations, non-unicast traffic,
	// and Bypass destinations are excluded before processing TCP/UDP.
	// The selected proxy's NoUDP, NoIPv6, and DNSHijack settings apply.
	//
	// cgtproxy does not change forwarding sysctls, interface settings, or other
	// firewall rules. The administrator must provide IP forwarding, firewall
	// permissions, and any NAT needed for direct forwarding. Docker usually
	// configures these. See the example configuration for loopback DNS setup.
	Gateway *Target `yaml:"gateway"`
	// Bypass lists destinations excluded from all TPROXY targets. When DNAT
	// changes a destination to one of these addresses, cgtproxy clears its proxy
	// mark on every packet so it follows normal routing. Marks belonging to
	// other network components are preserved.
	Bypass Bypass `yaml:"bypass" validate:"dive,ipv4|cidrv4|ipv6|cidrv6"`
	// TProxies defines the named transparent proxy servers available to Rules and Gateway.
	TProxies map[string]*TProxy `yaml:"tproxies" validate:"dive"`
	// Rules selects targets for local traffic by matching socket cgroup paths
	// in order. Packets without a queryable socket cgroup follow normal routing.
	// The first rule matching / selects the root target, inherited by local
	// descendant cgroups without a more specific target.
	// Both direct and tproxy targets are supported for the root.
	// A catch-all match such as /.* also matches /; use ^/$ to select the root
	// independently, including an explicit direct root target.
	Rules []Rule `yaml:"rules" validate:"dive"`
	// RouteTable identifies a dedicated table for TPROXY local default routes.
	// Valid IDs are 1-4294967295, excluding 253 (default), 254 (main), and 255 (local).
	// cgtproxy removes the routes it installs when it stops.
	RouteTable int `yaml:"route-table" validate:"gt=0,lte=4294967295,ne=253,ne=254,ne=255"`

	log *zap.SugaredLogger `yaml:"-"`
	raw []byte
}

// Bypass is a list of IPv4 or IPv6 addresses and CIDR ranges.
type Bypass []string

// CGroupRoot identifies the cgroup v2 mount path or requests AUTO discovery.
type CGroupRoot string

// Rule selects exactly one of Direct or TProxy for traffic from a cgroup.
type Rule struct {
	// Match is a regular expression matching a cgroup path relative to the
	// root of cgroupfs, including the leading slash.
	Match string `yaml:"match" validate:"required"`

	Target `yaml:",inline"`
}

// Target selects exactly one of direct: true or a named tproxy.
// Cgroup rules and gateway traffic use the same target format.
type Target struct {
	// TProxy selects a transparent proxy server defined in Config.TProxies.
	TProxy string `yaml:"tproxy" validate:"required_without=Direct,excluded_with=Direct"`
	// Direct leaves traffic on its normal route without proxying it.
	Direct bool `yaml:"direct" validate:"required_without=TProxy,excluded_with=TProxy"`
}

// TProxy describes a TPROXY server.
type TProxy struct {
	// Name is the nonempty tproxies map key. It generates nftables chains named
	// Name and Name+"-MARK", plus Name+"-DNS" when DNSHijack is configured.
	// Each generated name must be unique across proxies, differ from the internal
	// chain names in internal/types/chains.go, contain no NUL, and fit in 255 bytes.
	Name string `yaml:"-"`
	// NoUDP leaves UDP traffic direct, except for configured IPv4 DNS hijacking.
	NoUDP bool `yaml:"no-udp"`
	// NoIPv6 leaves IPv6 traffic direct.
	NoIPv6 bool `yaml:"no-ipv6"`
	// Port is the transparent proxy's listening port.
	Port uint16 `yaml:"port" validate:"required"`
	// Mark is the fire wall mark used to identify the TPROXY server
	// and trigger reroute operation of netfliter
	// from OUTPUT to PREROUTING internally.
	// It **NOT** means that this TPROXY server
	// must send traffic with the fwmark.
	// This mark is designed to be changeable for user
	// to make sure this mark is not conflict
	// with any fire wall mark in use.
	Mark FireWallMark `yaml:"mark" validate:"required"`
	// DNSHijack optionally redirects IPv4 DNS requests selected for this proxy
	// to a DNS server, for example for fake-IP operation. UDP destination port 53
	// is hijacked even with NoUDP; TCP port 53 is included when DNSHijack.TCP is
	// enabled. This applies to both cgroup and gateway targets.
	//
	// DNS DNAT runs before TPROXY and clears the proxy mark so packets are routed
	// to the DNS destination. Later packets skip marking only if their original
	// destination port is 53 and the translated destination matches this DNS
	// configuration. Other DNAT connections follow their cgroup or gateway target,
	// subject to Bypass.
	DNSHijack *DNSHijack `yaml:"dns-hijack"`
}

type FireWallMark uint32

// DNSHijack describes the IPv4 DNS server used for DNS redirection.
// Forwarding DNS to a loopback address requires route_localnet on the incoming
// interface. See the example configuration for commands and troubleshooting:
// https://github.com/black-desk/cgtproxy/blob/master/docs/troubleshooting.md#docker-container-dns-times-out-in-gateway-mode
type DNSHijack struct {
	// IP is the destination IPv4 address; it defaults to 127.0.0.1.
	IP *string `yaml:"ip" validate:"omitempty,ip4_addr"`
	// Port is required when DNS hijacking is configured; valid range: 1-65535.
	// Set 53 explicitly for a resolver listening on the standard DNS port.
	Port uint16 `yaml:"port" validate:"required"`
	// If TCP is set to true,
	// tcp traffic will be hijacked, too,
	// when it's destination port is 53.
	TCP bool `yaml:"tcp"`
}
