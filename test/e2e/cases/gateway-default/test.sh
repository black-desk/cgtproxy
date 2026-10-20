#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
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
