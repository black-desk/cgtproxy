package config

type Config struct {
	ConfigV1 `yaml:",inline"`
}

type ConfigV1 struct {
	Version  uint8     `yaml:"version" validate:"required,eq=1"`
	Repeater *Repeater `yaml:"repeater"`

	Proxies  map[string]*Proxy  `yaml:"proxies" validate:"dive"`
	TProxies map[string]*TProxy `yaml:"tproxies" validate:"dive"`

	Rules      []Rule     `yaml:"rules" validate:"dive"`
	Bypass     *Bypass    `yaml:"bypass"`
	CgroupRoot CgroupRoot `yaml:"cgroup-root" validate:"required,dirpath|eq=AUTO"`
	RouteTable int        `yaml:"route-table"`
	Marks      string     `yaml:"marks" validate:"required"`
}

type CgroupRoot string
type RerouteMark uint32

// Bypass describes the bypass rules apply to all the TPROXY servers.
// If the destination matched in Bypass, the traffic will not be touched.
type Bypass struct {
	IPV4 []string `yaml:"ipv4" validate:"dive,ipv4"`
	IPV6 []string `yaml:"ipv6" validate:"dive,ipv6"`
}

// Rule describes a rule about how to handle traffic comes from a cgroup.
type Rule struct {
	// Match is an regex expression
	// to match an cgroup path relative to the root of cgroupfs.
	Match string `yaml:"match" validate:"required"`

	// TProxy means that the traffic comes from this cgroup
	// should be redirected to a TPROXY server.
	TProxy string `yaml:"tproxy" validate:"required_without_all=Proxy Drop Direct,excluded_with=Proxy Drop Direct"`
	// Proxy means that the traffic comes from this cgroup
	// should be redirected to a proxy server.
	//
	// NOTE: This is not implemented yet.
	Proxy string `yaml:"proxy" validate:"required_without_all=TProxy Drop Direct,excluded_with=TProxy Drop Direct"`
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
	// NOTE: This field is not used yet.
	Addr *string `yaml:"addr" validate:"hostname|ip"`
	Port uint16  `yaml:"port" validate:"required"`
	// Mark is the fwmark used to identify the TPROXY server.
	// It **NOT** means that this TPROXY server
	// must send traffic with the fwmark.
	// This mark cgtproxy use internally designed to be changeable
	// to void fwmark confliction with other program using nftables.
	Mark RerouteMark `yaml:"mark"`
	// DNSHijack will hijack the dns request traffic
	// should redirect to this TPROXY server,
	// and send them to directory to a dns server described in DNSHijack.
	// This option is for fake-ip.
	DNSHijack *DNSHijack `yaml:"dns-hijack"`
}

type DNSHijack struct {
	Addr string `yaml:"addr" validate:"ip4_addr"`
	Port uint16 `yaml:"port"`
	// If TCP is set to true,
	// tcp traffic to any 53 port will be hijacked too.
	TCP bool `yaml:"tcp"`
}

// Repeater is configuration for a builtin TPROXY server,
// it is required if you have any entry in Proxies.
//
// NOTE: This is unimplemented yet.
type Repeater struct {
	// Listens is a list of ip which this TPROXY server will listen on.
	Listens []string `yaml:"listens" validate:"required,dive,ip"`
	// TProxyPorts is a string like [20000,21000)
	// describe a range of ports which this TPROXY server will use.
	TProxyPorts string `yaml:"tproxy-ports" validate:"required"`
}

// Proxy is describes a proxy server.
// If any of Proxy is configurated,
// the repeater is required to be configured too.
//
// NOTE: This is not implemented yet.
type Proxy struct {
	Protocol string `yaml:"protocol" validate:"required,eq=http|eq=https|eq=socks|eq=socks4|eq=socks5"`
	Addr     string `yaml:"addr" validate:"required,hostname|ip"`
	Port     uint16 `yaml:"port" validate:"required"`
	Auth     *Auth  `yaml:"auth"`
	UDP      bool   `yaml:"udp"`
	NoIPv6   bool   `yaml:"no-ipv6"`

	TProxy *TProxy `yaml:"-"`
}

// Auth describes a proxy server's authentication.
type Auth struct {
	User   string `yaml:"user" validate:"required"`
	Passwd string `yaml:"passwd" validate:"required"`
}
