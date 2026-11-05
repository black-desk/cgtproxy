#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
# An explicit direct gateway target keeps forwarding direct even when the root target proxies.
for network in tcp4 udp4; do
	probe forwarded "$network" 198.51.100.10:18080 origin direct
	probe forwarded "$network" 198.51.100.10:53 origin direct
	probe unmatched "$network" 198.51.100.10:18080 origin proxy
	probe proxied "$network" 198.51.100.10:18080 origin proxy
	probe proxied "$network" 198.51.100.10:53 dns direct 172.31.254.1:15353
done
for network in tcp6 udp6; do
	probe forwarded "$network" '[2001:db8::10]:18080' origin direct
	probe forwarded "$network" '[2001:db8::10]:53' origin direct
	probe unmatched "$network" '[2001:db8::10]:18080' origin proxy
	probe proxied "$network" '[2001:db8::10]:18080' origin proxy
done
