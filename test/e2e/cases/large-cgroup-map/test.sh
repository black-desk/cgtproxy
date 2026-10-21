#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# The combined entries exceed the 16-bit nested attribute length. Proxy entries
# add variable-length chain names, so a fixed element count is unsafe.
for ((i = 0; i < 300; i++)); do
	mkdir "/sys/fs/cgroup/group-proxy-$i"
done
for ((i = 0; i < 20; i++)); do
	mkdir "/sys/fs/cgroup/group-direct-$i"
done
start_cgtproxy

check_map() {
	"$FIXTURE_BIN" check-cgroup-map \
		-proxy-chain proxy-with-a-long-chain-name-to-exercise-variable-length-verdict-encoding-and-netlink-attribute-alignment-when-splitting-large-cgroup-maps-without-changing-the-transaction-boundaries-or-dropping-elements-MARK
}
check_map
for i in 0 10 19; do
	probe "group-direct-$i" tcp4 198.51.100.10:18080 origin
	probe "group-proxy-$((i * 15))" tcp4 198.51.100.10:18080 proxy
done
probe direct tcp4 198.51.100.10:18080 origin

# Reconcile a removal and recreation with the large map already installed.
# Check all keys again to catch stale kernel state after incremental updates.
rmdir /sys/fs/cgroup/group-direct-0
wait_until check_map
mkdir /sys/fs/cgroup/group-direct-0
wait_until check_map
probe group-direct-0 udp4 198.51.100.10:18080 origin
probe group-proxy-299 udp4 198.51.100.10:18080 proxy
if grep -q 'Failed to update cgroup routes' "$CASE_WORK/cgtproxy.log"; then
	echo 'Large cgroup map failed to reconcile' >&2
	exit 1
fi
