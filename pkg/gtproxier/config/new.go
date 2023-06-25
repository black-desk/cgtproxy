package config

import (
	. "github.com/black-desk/lib/go/errwrap"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Opt func(c *Config) (ret *Config, err error)

func New(opts ...Opt) (ret *Config, err error) {
	defer Wrap(&err, "load configuration")

	c := &Config{}
	for i := range opts {
		c, err = opts[i](c)
		if err != nil {
			return
		}
	}

	if c.log == nil {
		c.log = zap.NewNop().Sugar()
	}

	err = yaml.Unmarshal(c.raw, c)
	if err != nil {
		Wrap(&err, "unmarshal configuration")
		return
	}

	err = c.check()
	if err != nil {
		return
	}

	ret = c
	return
}

func WithContent(raw []byte) Opt {
	return func(c *Config) (ret *Config, err error) {
		c.raw = raw
		ret = c
		return
	}
}

func WithLogger(log *zap.SugaredLogger) Opt {
	return func(c *Config) (ret *Config, err error) {
		c.log = log
		ret = c
		return
	}
}
