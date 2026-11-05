#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Existing cgroups are found by the initial scan.
mkdir /sys/fs/cgroup/proxied/{direct,second}
start_cgtproxy
for network in tcp4 udp4; do
	probe proxied "$network" 198.51.100.10:18080 origin proxy
	probe direct "$network" 198.51.100.10:18080 origin direct
	probe unmatched "$network" 198.51.100.10:18080 origin direct
	probe proxied "$network" 198.51.100.20:18080 origin direct
done
probe proxied tcp6 '[2001:db8::10]:18080' origin proxy
probe proxied udp6 '[2001:db8::10]:18080' origin proxy
probe proxied tcp6 '[2001:db8::20]:18080' origin direct
probe proxied/second tcp4 198.51.100.10:18080 origin second
# First matching rule wins over the later proxied-parent rule.
probe proxied/direct tcp4 198.51.100.10:18080 origin direct
# Create, delete and recreate the same cgroup while cgtproxy is running.
mkdir /sys/fs/cgroup/proxied/live
wait_until probe proxied/live tcp4 198.51.100.10:18080 origin proxy
rmdir /sys/fs/cgroup/proxied/live
mkdir /sys/fs/cgroup/proxied/live
wait_until probe proxied/live udp4 198.51.100.10:18080 origin proxy
# Local DNS is handled through OUTPUT NAT, including successive requests.
probe proxied udp4 198.51.100.10:53 dns direct 172.31.254.1:15353
probe proxied tcp4 198.51.100.10:53 dns direct 172.31.254.1:15353
