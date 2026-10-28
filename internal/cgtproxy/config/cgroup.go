// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"os"

	. "github.com/black-desk/cgtproxy/internal/errwrap"
)

func getCgroupRoot() (cgroupRoot CGroupRoot, err error) {
	defer Wrap(&err, "get cgroupv2 mount point")

	paths := []string{"/sys/fs/cgroup/unified", "/sys/fs/cgroup"}

	for i := range paths {
		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(paths[i])
		if err != nil {
			continue
		}

		if !fileInfo.IsDir() {
			continue
		}

		cgroupRoot = CGroupRoot(paths[i])
		return
	}

	err = ErrCannotFoundCgroupv2Root
	return
}
