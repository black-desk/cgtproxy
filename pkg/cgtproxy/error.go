// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgtproxy

import (
	"errors"
)

var (
	ErrConfigMissing        = errors.New("config is missing.")
	ErrLoggerMissing        = errors.New("logger is missing.")
	ErrCGroupMonitorMissing = errors.New("cgroup monitor is missing.")
	ErrRouteManagerMissing  = errors.New("route manager is missing.")
)
