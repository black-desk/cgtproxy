#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Failing to create our table must neither change nor delete an existing table.
nft -f - <<'NFT'
table inet cgtproxy {
    chain sentinel {
        counter accept
    }
}
NFT
if (LOG_LEVEL=debug in_group manager "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
	--log-file "$CASE_WORK/cgtproxy.log"); then
	echo 'Initialization unexpectedly succeeded with an existing table' >&2
	exit 1
fi
nft list chain inet cgtproxy sentinel | grep -q 'counter packets 0 bytes 0 accept'
if nft list chain inet cgtproxy output-mangle >/dev/null 2>&1; then
	echo 'Failed initialization left a partial ruleset' >&2
	exit 1
fi
if {
	ip rule show
	ip -6 rule show
} | grep -q 'lookup 31300'; then
	echo 'Failed initialization left policy rules behind' >&2
	exit 1
fi
[[ -z $(ip route show table 31300 2>/dev/null) && -z $(ip -6 route show table 31300 2>/dev/null) ]]
nft delete table inet cgtproxy
probe unmatched tcp4 198.51.100.10:18080 origin
