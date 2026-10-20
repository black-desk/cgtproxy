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
# With gateway omitted, only local traffic uses the root proxy and its DNS.
for network in tcp4 udp4; do
	probe forwarded "$network" 198.51.100.10:18080 origin
	probe forwarded "$network" 198.51.100.10:53 origin
	probe unmatched "$network" 198.51.100.10:18080 proxy
	probe unmatched "$network" 198.51.100.10:53 dns
done
for network in tcp6 udp6; do
	probe forwarded "$network" '[2001:db8::10]:18080' origin
	probe forwarded "$network" '[2001:db8::10]:53' origin
	probe unmatched "$network" '[2001:db8::10]:18080' proxy
	probe unmatched "$network" '[2001:db8::10]:53' proxy
done
