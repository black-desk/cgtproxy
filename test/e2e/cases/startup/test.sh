#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

mkdir /sys/fs/cgroup/proxied/initial

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
wait "$churn_pid"
probe proxied/initial tcp4 198.51.100.10:18080 origin
probe unmatched tcp4 198.51.100.10:18080 proxy
probe forwarded udp4 198.51.100.10:18080 proxy
# Verify traffic originating in the root cgroup as well as its descendants.
probe / tcp4 198.51.100.10:18080 proxy
probe / udp4 198.51.100.10:18080 proxy
probe / tcp6 '[2001:db8::10]:18080' proxy
probe / udp6 '[2001:db8::10]:18080' proxy

# Queue a removal and recreation before the manager can consume either event.
mkdir /sys/fs/cgroup/proxied/live
wait_until probe proxied/live tcp4 198.51.100.10:18080 origin
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
wait_until probe proxied/live tcp4 198.51.100.10:18080 origin
wait_until probe proxied/live udp4 198.51.100.10:18080 origin
rmdir /sys/fs/cgroup/proxied/live
if grep -q 'Failed to update cgroup routes' "$CASE_WORK/cgtproxy.log"; then
	echo 'Queued cgroup events failed to reconcile' >&2
	exit 1
fi
