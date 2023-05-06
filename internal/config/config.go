package config

// NOTE(black_desk): Always call Check before use Config.
type Config struct {
	ConfigV1
}

type ConfigV1 struct {
	Version    uint8     `yaml:"version" validate:"required,eq=1"`
	Repeater   *Repeater `yaml:"repeater"`
	Rules      []Rule    `yaml:"rules" validate:"dive"`
	Bypass     *Bypass   `yaml:"bypass"`
	CgroupRoot string    `yaml:"cgroup-root" validate:"required,dirpath|eq=AUTO"`
	RouteTable int       `yaml:"route-table"`
	Mark       uint32    `yaml:"mark"`

	Proxies  map[*Proxy]struct{}  `yaml:"-"`
	TProxies map[*TProxy]struct{} `yaml:"-"`
}

type Bypass struct {
	IPV4 []string `yaml:"ipv4" validate:"dive,ipv4"`
	IPV6 []string `yaml:"ipv6" validate:"dive,ipv6"`
}

type Rule struct {
	Match string `yaml:"match" validate:"required"`

	TProxy *TProxy `yaml:"tproxy" validate:"required_without_all=Proxy Drop Direct,excluded_with=Proxy Drop Direct"`
	Proxy  *Proxy  `yaml:"proxy" validate:"required_without_all=TProxy Drop Direct,excluded_with=TProxy Drop Direct"`
	Drop   bool    `yaml:"drop" validate:"required_without_all=TProxy Proxy Direct,excluded_with=TProxy Proxy Direct"`
	Direct bool    `yaml:"direct" validate:"required_without_all=TProxy Proxy Drop,excluded_with=TProxy Proxy Drop"`
}

type TProxy struct {
	Name   string  `yaml:"name" validate:"required"` // use as chain name
	NoUDP  bool    `yaml:"no-udp"`
	NoIPv6 bool    `yaml:"no-ipv6"`
	Addr   *string `yaml:"addr" validate:"hostname"`
	Port   uint16  `yaml:"port" validate:"required"`
}

type Repeater struct {
	Listens     []string `yaml:"listens" validate:"required,dive,ip"`
	TProxyPorts string   `yaml:"tproxy-ports"`
}

type Proxy struct {
	Name     string `yaml:"name"`
	Protocol string `yaml:"protocol" validate:"required,eq=http|eq=https|eq=socks|eq=socks4|eq=socks5"`
	Addr     string `yaml:"addr" validate:"required,hostname|ip"`
	Port     uint16 `yaml:"port" validate:"required"`
	Auth     *Auth  `yaml:"auth"`
	UDP      bool   `yaml:"udp"`
	NoIPv6   bool   `yaml:"no-ipv6"`

	TProxy *TProxy `yaml:"-"`
}

type Auth struct {
	User   string `yaml:"user" validate:"required"`
	Passwd string `yaml:"passwd" validate:"required"`
}