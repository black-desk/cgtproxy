#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

mkdir /sys/fs/cgroup/proxied/initial
"$FIXTURE_BIN" check-nft-init -ready "$CASE_WORK/monitor.ready" -cgroup /sys/fs/cgroup/servers &
monitor_pid=$!
pids+=("$monitor_pid")
wait_until test -f "$CASE_WORK/monitor.ready"

# Exercise changes overlapping the initial scan and transaction construction.
(
	for ((i = 0; i < 32; i++)); do
		mkdir /sys/fs/cgroup/proxied/transient
		sleep 0.01
		rmdir /sys/fs/cgroup/proxied/transient
	done
) &
churn_pid=$!
pids+=("$churn_pid")
start_cgtproxy
wait "$monitor_pid"
wait "$churn_pid"
wait_until no_cgroup proxied/transient
probe proxied/initial tcp4 198.51.100.10:18080 origin
probe unmatched tcp4 198.51.100.10:18080 proxy
probe forwarded udp4 198.51.100.10:18080 proxy

# Queue a removal and recreation before the manager can consume either event.
mkdir /sys/fs/cgroup/proxied/live
wait_until has_cgroup proxied/live
old_inode=$(stat -c %i /sys/fs/cgroup/proxied/live)
manager_pid=${cgtproxy_pid:?}
kill -STOP "$manager_pid"
wait_until grep -Eq '^State:[[:space:]]+T' "/proc/$manager_pid/status"
rmdir /sys/fs/cgroup/proxied/live
mkdir /sys/fs/cgroup/proxied/live
[[ $old_inode != "$(stat -c %i /sys/fs/cgroup/proxied/live)" ]]
mkdir /sys/fs/cgroup/proxied/transient
rmdir /sys/fs/cgroup/proxied/transient
kill -CONT "$manager_pid"
wait_until has_cgroup proxied/live
probe proxied/live tcp4 198.51.100.10:18080 origin
probe proxied/live udp4 198.51.100.10:18080 origin
rmdir /sys/fs/cgroup/proxied/live
wait_until no_cgroup proxied/live
wait_until no_cgroup proxied/transient
if grep -q 'Failed to update cgroup routes' "$CASE_WORK/cgtproxy.log"; then
	echo 'Queued cgroup events failed to reconcile' >&2
	exit 1
fi
