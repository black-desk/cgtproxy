// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/black-desk/lib/go/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newCommandLogger(path string) (*zap.SugaredLogger, func(), error) {
	if path == "" {
		return logger.Get("cgtproxy"), func() {}, nil
	}

	level := zap.NewAtomicLevelAt(zap.InfoLevel)
	if value := os.Getenv("LOG_LEVEL"); value != "" {
		if err := level.UnmarshalText([]byte(value)); err != nil {
			return nil, nil, fmt.Errorf("invalid LOG_LEVEL: %w", err)
		}
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file %q: %w", path, err)
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.Lock(file), level,
	)
	log := zap.New(core, zap.AddStacktrace(zap.ErrorLevel)).Named("cgtproxy").Sugar()
	return log, func() {
		_ = log.Sync()
		_ = file.Close()
	}, nil
}
