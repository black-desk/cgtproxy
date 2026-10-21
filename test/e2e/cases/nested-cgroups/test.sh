#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
manager_pid=${cgtproxy_pid:?}

# Only the root is watched when these directories are created. Pausing the
# manager makes the missing recursive-watch window deterministic.
kill -STOP "$manager_pid"
wait_until grep -Eq '^State:[[:space:]]+T' "/proc/$manager_pid/status"
mkdir -p /sys/fs/cgroup/new/blocked/inner
kill -CONT "$manager_pid"
wait_until has_cgroup new
wait_until has_cgroup new/blocked
wait_until has_cgroup new/blocked/inner
probe new tcp4 198.51.100.10:18080 origin
probe new/blocked tcp4 198.51.100.10:18080 drop
probe new/blocked/inner udp4 198.51.100.10:18080 drop

# Scanned descendants must themselves be watched for subsequent changes.
mkdir /sys/fs/cgroup/new/blocked/inner/later
wait_until has_cgroup new/blocked/inner/later
probe new/blocked/inner/later tcp4 198.51.100.10:18080 drop
rmdir /sys/fs/cgroup/new/blocked/inner/later
wait_until no_cgroup new/blocked/inner/later

# Reconcile queued removals and recreation of a whole subtree at the same path.
kill -STOP "$manager_pid"
wait_until grep -Eq '^State:[[:space:]]+T' "/proc/$manager_pid/status"
rmdir /sys/fs/cgroup/new/blocked/inner /sys/fs/cgroup/new/blocked /sys/fs/cgroup/new
mkdir -p /sys/fs/cgroup/new/blocked/recreated
kill -CONT "$manager_pid"
wait_until no_cgroup new/blocked/inner
wait_until has_cgroup new/blocked/recreated
probe new/blocked/recreated tcp4 198.51.100.10:18080 drop
mkdir /sys/fs/cgroup/new/blocked/recreated/later
wait_until has_cgroup new/blocked/recreated/later
probe new/blocked/recreated/later udp4 198.51.100.10:18080 drop

rmdir /sys/fs/cgroup/new/blocked/recreated/later /sys/fs/cgroup/new/blocked/recreated \
	/sys/fs/cgroup/new/blocked /sys/fs/cgroup/new
wait_until no_cgroup new
wait_until no_cgroup new/blocked
wait_until no_cgroup new/blocked/recreated
wait_until no_cgroup new/blocked/recreated/later
if grep -q 'Failed to update cgroup routes' "$CASE_WORK/cgtproxy.log"; then
	echo 'Nested cgroup routes failed to reconcile' >&2
	exit 1
fi
