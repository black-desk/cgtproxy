package config

import (
	. "github.com/black-desk/lib/go/errwrap"
	"gopkg.in/yaml.v3"
)

func Load(content []byte) (ret *Config, err error) {
	defer Wrap(&err, "Failed to load configuration")

	cfg := &Config{}

	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		Wrap(&err, "Failed to unmarshal configuration.")
		return
	}

	err = cfg.check()
	if err != nil {
		return
	}

	ret = cfg
	return

}
