// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"errors"
)

var (
	ErrCannotFoundCgroupv2Root = errors.New("`cgroup2` mount point not found.")
)
