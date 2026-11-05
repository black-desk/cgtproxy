// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"bytes"
	"fmt"

	"github.com/black-desk/cgtproxy/internal/errwrap"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Opt func(c *Config) (ret *Config, err error)

func New(opts ...Opt) (ret *Config, err error) {
	defer errwrap.Wrap(&err, "load configuration")

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

	// Check the format before decoding its fields so legacy options such as
	// drop do not hide the migration advice behind an unknown-field error.
	var header struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(c.raw, &header); err != nil {
		return nil, fmt.Errorf("read configuration version: %w", err)
	}
	switch header.Version {
	case "2":
	case "1":
		return nil, fmt.Errorf("configuration version 1 is no longer supported: " +
			"migrate to version 2 by removing drop rules (use a separate firewall if blocking is needed), " +
			"setting an explicit nonzero port for each configured dns-hijack, " +
			"and changing version to 2; then run cgtproxy check config before restarting the service")
	case "":
		return nil, fmt.Errorf("configuration version is required; supported version: 2")
	default:
		return nil, fmt.Errorf("unsupported configuration version %q; supported version: 2", header.Version)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(c.raw))
	decoder.KnownFields(true)
	err = decoder.Decode(c)
	if err != nil {
		return nil, fmt.Errorf("unmarshal configuration: %w", err)
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
