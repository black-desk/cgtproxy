// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package logger

import (
	"bytes"
	"os"
	"text/template"
	"time"

	. "github.com/black-desk/lib/go/errwrap"
	"go.uber.org/zap"
)

const (
	defaultTestLogFilePathTemplate = "/tmp/io.github.black-desk.cgtproxy-test/log-{{.TIMESTAMP}}.txt"
)

func genLogFilePathFromTemplate(templStr string) (ret string, err error) {
	defer Wrap(&err, "gen log file path from template string")
	templ := template.New("test log file path")
	templ, err = templ.Parse(templStr)
	if err != nil {
		return
	}

	buf := new(bytes.Buffer)
	err = templ.Execute(buf, map[string]string{"TIMESTAMP": time.Now().String()})
	if err != nil {
		return
	}

	ret = buf.String()
	return
}

func ProvideLogger() (ret *zap.SugaredLogger, err error) {
	defer Wrap(&err, "create a logger for test")

	logFilePath := os.Getenv("CGTPROXY_TEST_LOGFILE")
	if logFilePath == "" {
		logFilePath, err = genLogFilePathFromTemplate(defaultTestLogFilePathTemplate)
		if err != nil {
			return
		}
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = []string{logFilePath}

	var logger *zap.Logger
	logger, err = cfg.Build()
	if err != nil {
		Wrap(&err, "build zap logger from development config")
		return
	}

	ret = logger.Sugar()
	return
}
