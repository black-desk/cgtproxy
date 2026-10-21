// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package types

// Shared with configuration validation so proxy chains cannot reuse these names.
const (
	ChainCgroupRoot    = "cgroup-root"
	ChainOutputMangle  = "output-mangle"
	ChainOutputNAT     = "output-nat"
	ChainPrerouting    = "prerouting"
	ChainForwardMangle = "forward-mangle"
	ChainPreroutingNAT = "prerouting-nat"
)
