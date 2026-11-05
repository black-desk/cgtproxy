#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
for network in tcp4 udp4; do
	probe forwarded "$network" 198.51.100.10:18080 origin proxy
	probe forwarded "$network" 198.51.100.20:18080 origin direct
	probe forwarded "$network" 198.51.100.10:53 dns direct 172.31.254.1:15353
done
for network in tcp6 udp6; do
	probe forwarded "$network" '[2001:db8::10]:18080' origin proxy
	probe forwarded "$network" '[2001:db8::20]:18080' origin direct
	probe forwarded "$network" '[2001:db8::10]:53' origin proxy
done
probe forwarded tcp4 172.31.254.1:18081 host direct
probe direct tcp4 198.51.100.10:18080 origin direct

# Keep sending on one UDP connection to exercise established proxy and DNS sessions.
for network in udp4 udp6; do
	address=198.51.100.10:18080
	if [[ $network == udp6 ]]; then address='[2001:db8::10]:18080'; fi
	probe forwarded "$network" "$address" origin proxy "$address" 256
done
probe forwarded udp4 198.51.100.10:53 dns direct 172.31.254.1:15353 256

# Local root and unmatched cgroups remain direct while the gateway proxies.
probe / tcp4 198.51.100.10:18080 origin direct
probe unmatched udp4 198.51.100.10:18080 origin direct
