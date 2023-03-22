package config

type Config = ConfigV1

type ConfigV1 struct {
	Version    uint8     `yaml:"version" validate:"required,eq=1"`
	Repeater   *Repeater `yaml:"repeater"`
	Rules      []Rule    `yaml:"rules" validate:"dive"`
	CgroupRoot string    `yaml:"cgroup-root" validate:"required,dirpath"`
}

type Rule struct {
	Match string `yaml:"match" validate:"required"`

	TProxy *TProxy   `yaml:"tproxy" validate:"required_without_all=Proxy Redir,excluded_with=Redir Proxy"`
	Redir  *Redirect `yaml:"redir" validate:"required_without_all=TProxy Proxy,excluded_with=TProxy Proxy"`
	Proxy  *Proxy    `yaml:"proxy" validate:"required_without_all=TProxy Redir,excluded_with=TProxy Redir"`

	Mark *string `yaml:"mask"`
}

type Redirect struct {
	Ports string `yaml:"ports" validate:"required"`
}

type TProxy struct {
	Addr *string `yaml:"addr" validate:"hostname"`
	Port uint16  `yaml:"port" validate:"required"`
}

type Repeater struct {
	Listens    []string `yaml:"listens" validate:"required,dive,ip"`
	TProxyPort uint16   `yaml:"tproxy-port" validate:"required"`
}

type Proxy struct {
	Protocol string `yaml:"protocol" validate:"required,eq=http|eq=https|eq=socks|eq=socks4|eq=socks5"`
	Addr     string `yaml:"addr" validate:"required,hostname"`
	Port     uint16 `yaml:"port" validate:"required"`
	Auth     *Auth  `yaml:"auth"`
}

type Auth struct {
	User   string `yaml:"user" validate:"required"`
	Passwd string `yaml:"passwd" validate:"required"`
}
