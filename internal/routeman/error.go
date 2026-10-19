// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import "errors"

var (
	ErrNFTManagerMissing      = errors.New("nft manager is missing.")
	ErrConfigMissing          = errors.New("config is missing.")
	ErrCGroupEventChanMissing = errors.New("cgroup event channel is missing.")
)
