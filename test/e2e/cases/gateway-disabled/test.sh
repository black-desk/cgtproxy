#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
nft list chain inet cgtproxy forward-mangle >"$CASE_WORK/forward-mangle.nft"
if grep -q 'goto cgroup-root' "$CASE_WORK/forward-mangle.nft"; then
	echo 'Gateway mode is disabled but forwarding uses the root target' >&2
	exit 1
fi
# An explicit false keeps forwarding direct even when the root target drops.
for network in tcp4 udp4; do
	probe forwarded "$network" 198.51.100.10:18080 origin
	probe forwarded "$network" 198.51.100.10:53 origin
	probe unmatched "$network" 198.51.100.10:18080 drop
	probe proxied "$network" 198.51.100.10:18080 proxy
	probe proxied "$network" 198.51.100.10:53 dns
done
for network in tcp6 udp6; do
	probe forwarded "$network" '[2001:db8::10]:18080' origin
	probe forwarded "$network" '[2001:db8::10]:53' origin
	probe unmatched "$network" '[2001:db8::10]:18080' drop
	probe proxied "$network" '[2001:db8::10]:18080' proxy
done
