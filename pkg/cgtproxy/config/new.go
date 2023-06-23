package config

import (
	. "github.com/black-desk/lib/go/errwrap"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func Load(content []byte, log *zap.SugaredLogger) (ret *Config, err error) {
	defer Wrap(&err, "load configuration")

	cfg := &Config{}
	cfg.log = log
	if cfg.log == nil {
		cfg.log = zap.NewNop().Sugar()
	}

	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		Wrap(&err, "unmarshal configuration")
		return
	}

	err = cfg.check()
	if err != nil {
		return
	}

	ret = cfg
	return
}
