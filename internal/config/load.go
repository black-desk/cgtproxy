package config

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	"gopkg.in/yaml.v3"
)

func Load(content []byte) (ret *Config, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Capture()+
			"Failed to load configuration:\n%w",
			err,
		)
	}()

	cfg := &Config{}

	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		err = fmt.Errorf(location.Capture()+
			"Failed to unmarshal configuration:\n%w", err)
		return
	}

	err = cfg.check()
	if err != nil {
		return
	}

	ret = cfg

	return
}
