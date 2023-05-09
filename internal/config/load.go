package config

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/location"
	"gopkg.in/yaml.v3"
)

func Load(content []byte) (ret *Config, err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Catch()+
			"Failed to load configuration:\n%w",
			err,
		)
	}()

	cfg := &Config{}

	if err = yaml.Unmarshal(content, cfg); err != nil {
		err = fmt.Errorf(location.Catch()+
			"Failed to unmarshal configuration:\n%w", err)
		return
	}

	if err = cfg.check(); err != nil {
		return
	}

	ret = cfg

	return
}
