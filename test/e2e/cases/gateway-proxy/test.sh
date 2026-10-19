#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
for network in tcp4 udp4; do
	probe forwarded "$network" 198.51.100.10:18080 proxy
	probe forwarded "$network" 198.51.100.20:18080 origin
	probe forwarded "$network" 198.51.100.10:53 dns
done
for network in tcp6 udp6; do
	probe forwarded "$network" '[2001:db8::10]:18080' proxy
	probe forwarded "$network" '[2001:db8::20]:18080' origin
	probe forwarded "$network" '[2001:db8::10]:53' proxy
done
probe forwarded tcp4 172.31.254.1:18081 host
probe direct tcp4 198.51.100.10:18080 origin
