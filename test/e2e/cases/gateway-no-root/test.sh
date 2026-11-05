#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
for network in tcp4 udp4; do probe forwarded "$network" 198.51.100.10:18080 origin proxy; done
for network in tcp6 udp6; do probe forwarded "$network" '[2001:db8::10]:18080' origin proxy; done
probe proxied tcp4 198.51.100.10:18080 origin proxy
probe proxied udp6 '[2001:db8::10]:18080' origin proxy

# The gateway proxy does not create a target for unmatched local cgroups.
probe unmatched tcp4 198.51.100.10:18080 origin direct
probe / udp4 198.51.100.10:18080 origin direct
probe forwarded udp4 198.51.100.10:53 dns direct 172.31.254.1:15353
