//go:build debug

// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func commandLoggerConfig() (zapcore.EncoderConfig, []zap.Option, zapcore.Level) {
	return zap.NewDevelopmentEncoderConfig(), []zap.Option{
		zap.Development(),
		zap.AddStacktrace(zap.WarnLevel),
	}, zap.DebugLevel
}
