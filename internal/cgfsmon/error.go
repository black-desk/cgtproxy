// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgfsmon

import "errors"

var (
	ErrContextMissing         = errors.New("context is missing.")
	ErrCGroupRootNotFound     = errors.New("cgroup v2 file system mount point is missing.")
	ErrLoggerMissing          = errors.New("logger is missing.")
	ErrUnderlingWatcherExited = errors.New("underling file system watcher has exited.")
)
