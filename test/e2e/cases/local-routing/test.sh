#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Existing cgroups are found by the initial scan.
mkdir /sys/fs/cgroup/proxied/{direct,drop,second}
start_cgtproxy
for network in tcp4 udp4; do
	probe proxied "$network" 198.51.100.10:18080 proxy
	probe direct "$network" 198.51.100.10:18080 origin
	probe unmatched "$network" 198.51.100.10:18080 origin
	probe blocked "$network" 198.51.100.10:18080 drop
	probe proxied "$network" 198.51.100.20:18080 origin
done
probe proxied tcp6 '[2001:db8::10]:18080' proxy
probe proxied udp6 '[2001:db8::10]:18080' proxy
probe proxied tcp6 '[2001:db8::20]:18080' origin
probe proxied/second tcp4 198.51.100.10:18080 second
probe proxied/direct tcp4 198.51.100.10:18080 origin
# First matching rule wins over the later proxied-parent rule.
probe proxied/drop tcp4 198.51.100.10:18080 drop
# Create, delete and recreate the same cgroup while cgtproxy is running.
mkdir /sys/fs/cgroup/proxied/live
wait_until has_cgroup proxied/live
probe proxied/live tcp4 198.51.100.10:18080 proxy
rmdir /sys/fs/cgroup/proxied/live
wait_until no_cgroup proxied/live
mkdir /sys/fs/cgroup/proxied/live
wait_until has_cgroup proxied/live
probe proxied/live udp4 198.51.100.10:18080 proxy
# Local DNS is handled through OUTPUT NAT, including successive requests.
probe proxied udp4 198.51.100.10:53 dns
probe proxied tcp4 198.51.100.10:53 dns
