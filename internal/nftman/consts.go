// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

const (
	NftTableName = "cgtproxy"

	// nftables v0.3.0 encodes NFTA_CT_DIRECTION as a big-endian uint32,
	// while Linux reads its first byte as uint8. Put IP_CT_DIR_REPLY there.
	ctReplyDirection = 1 << 24
)
