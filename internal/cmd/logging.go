// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/black-desk/zap-journal/conn"
	"github.com/black-desk/zap-journal/encoder"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(path string) (*zap.SugaredLogger, func(), error) {
	cfg, options, defaultLevel := loggerConfig()

	env := os.Getenv("LOG_LEVEL")
	if env == "" {
		env = defaultLevel.String()
	}

	level := zap.NewAtomicLevel()
	err := level.UnmarshalText([]byte(env))
	if err != nil {
		return nil, nil, fmt.Errorf(
			"invalid LOG_LEVEL: %w",
			err,
		)
	}

	if path != "" {
		return newFileLogger(path, cfg, options, level)
	}

	var cores []zapcore.Core
	info, err := os.Stderr.Stat()

	if err == nil && info.Mode()&os.ModeCharDevice != 0 {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zapcore.Lock(os.Stderr),
			level,
		))
	}

	journalCfg := cfg
	journalCfg.StacktraceKey = "STACKTRACE"
	enc, err := encoder.New(encoder.WithCfg(journalCfg))

	var journal *conn.JournalConn
	if err == nil {
		journal, err = conn.New()
	}
	if err != nil {
		log.Printf("Failed to create journal zapcore: %s.", err)
	} else {
		cores = append(cores, zapcore.NewCore(enc, journal, level))
	}

	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg),
			zapcore.Lock(os.Stderr), level,
		))
	}
	logger := zap.New(
		zapcore.NewTee(cores...),
		options...,
	).Named("cgtproxy").Sugar()
	return logger, func() {
		_ = logger.Sync()
		if journal != nil {
			_ = journal.Close()
		}
	}, nil
}

func newFileLogger(
	path string,
	cfg zapcore.EncoderConfig,
	options []zap.Option,
	level zapcore.LevelEnabler,
) (
	*zap.SugaredLogger, func(), error) {
	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0600,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file %q: %w", path, err)
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.Lock(file),
		level,
	)
	log := zap.New(core, options...).Named("cgtproxy").Sugar()
	return log, func() {
		_ = log.Sync()
		_ = file.Close()
	}, nil
}
