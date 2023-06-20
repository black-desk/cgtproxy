package config

type Config struct {
	ConfigV1 `yaml:",inline"`
}

type ConfigV1 struct {
	Version uint8 `yaml:"version" validate:"required,eq=1"`

	CgroupRoot CgroupRoot         `yaml:"cgroup-root" validate:"required,dirpath|eq=AUTO"`
	Bypass     *Bypass            `yaml:"bypass"`
	TProxies   map[string]*TProxy `yaml:"tproxies" validate:"dive"`
	Rules      []Rule             `yaml:"rules" validate:"dive"`
	// The route table number cgtproxy will create to route TPROXY traffic.
	// This table will be removed when cgtproxy stopped.
	RouteTable int `yaml:"route-table" validate:"required"`
}

type CgroupRoot string
type FireWallMark uint32

// Bypass describes the bypass rules apply to all the TPROXY servers.
// If the destination matched in Bypass, the traffic will not be touched.
type Bypass struct {
	IPV4 []string `yaml:"ipv4" validate:"dive,ipv4|cidrv4"`
	IPV6 []string `yaml:"ipv6" validate:"dive,ipv6|cidrv6"`
}

// Rule describes a rule about how to handle traffic comes from a cgroup.
type Rule struct {
	// Match is an regex expression
	// to match an cgroup path relative to the root of cgroupfs.
	Match string `yaml:"match" validate:"required"`

	// TProxy means that the traffic comes from this cgroup
	// should be redirected to a TPROXY server.
	TProxy string `yaml:"tproxy" validate:"required_without_all=Proxy Drop Direct,excluded_with=Proxy Drop Direct"`
	// Drop means that the traffic comes from this cgroup will be dropped.
	Drop bool `yaml:"drop" validate:"required_without_all=TProxy Proxy Direct,excluded_with=TProxy Proxy Direct"`
	// Direct means that the traffic comes from this cgroup will not be touched.
	Direct bool `yaml:"direct" validate:"required_without_all=TProxy Proxy Drop,excluded_with=TProxy Proxy Drop"`
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

type DNSHijack struct {
	IP   *string `yaml:"ip" validate:"ip4_addr"`
	Port uint16  `yaml:"port"`
	// If TCP is set to true,
	// tcp traffic to any 53 port will be hijacked too.
	TCP bool `yaml:"tcp"`
}
