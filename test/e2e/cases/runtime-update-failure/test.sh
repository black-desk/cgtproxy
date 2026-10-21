#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

mkdir /sys/fs/cgroup/proxied/gone
start_cgtproxy
probe proxied tcp4 198.51.100.10:18080 proxy

# Simulate an unexpected external edit. Removing the cgroup now attempts to
# delete a missing map element, so the incremental transaction must fail.
nft flush map inet cgtproxy cgroup-vmap
rmdir /sys/fs/cgroup/proxied/gone
daemon_exited() { ! kill -0 "$cgtproxy_pid" 2>/dev/null; }
wait_until daemon_exited
status=0
wait "$cgtproxy_pid" || status=$?
cgtproxy_pid=
if [[ $status != 1 ]]; then
	echo "Expected route update failure, got exit status $status" >&2
	exit 1
fi
grep -q 'update .*routes in nftable' "$CASE_WORK/cgtproxy.log"
assert_network_cleanup
probe proxied tcp4 198.51.100.10:18080 origin
probe proxied udp6 '[2001:db8::10]:18080' origin
