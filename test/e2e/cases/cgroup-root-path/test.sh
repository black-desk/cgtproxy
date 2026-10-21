#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

mkdir /sys/fs/cgroup/proxied/initial
start_cgtproxy
for network in tcp4 udp4; do
	probe proxied/initial "$network" 198.51.100.10:18080 proxy
	probe direct "$network" 198.51.100.10:18080 origin
	probe blocked "$network" 198.51.100.10:18080 drop
done
mkdir /sys/fs/cgroup/proxied/live
wait_until has_cgroup proxied/live
probe proxied/live tcp4 198.51.100.10:18080 proxy
rmdir /sys/fs/cgroup/proxied/live
wait_until no_cgroup proxied/live
stop_cgtproxy

# ".." and repeated separators must resolve to the same cgroup ancestry.
sed 's|^cgroup-root:.*|cgroup-root: /sys/fs/cgroup/../cgroup//|' "$CASE_DIR/config.yaml" >"$CASE_WORK/config.yaml"
CASE_DIR=$CASE_WORK
start_cgtproxy
probe blocked tcp4 198.51.100.10:18080 drop
probe proxied/initial udp4 198.51.100.10:18080 proxy
