package config

import "go.uber.org/zap"

type Config struct {
	Version string   `yaml:"version" validate:"required,eq=1"`
	Proxies []Proxy  `yaml:"proxies" validate:"dive"`
	Listens []string `yaml:"listens" validate:"dive,ip"`

	log *zap.SugaredLogger
	raw []byte
}

type Proxy struct {
	URL        string `yaml:"url" validate:"required"`
	TProxyPort uint16 `yaml:"tproxy-port" validate:"required"`
}
