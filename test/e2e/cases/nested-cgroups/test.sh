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
mkdir -p /sys/fs/cgroup/new/proxied/inner
kill -CONT "$manager_pid"
wait_until probe new tcp4 198.51.100.10:18080 origin direct
wait_until probe new/proxied tcp4 198.51.100.10:18080 origin proxy
wait_until probe new/proxied/inner udp4 198.51.100.10:18080 origin proxy

# Scanned descendants must themselves be watched for subsequent changes.
mkdir /sys/fs/cgroup/new/proxied/inner/later
wait_until probe new/proxied/inner/later tcp4 198.51.100.10:18080 origin proxy
rmdir /sys/fs/cgroup/new/proxied/inner/later

# Reconcile queued removals and recreation of a whole subtree at the same path.
kill -STOP "$manager_pid"
wait_until grep -Eq '^State:[[:space:]]+T' "/proc/$manager_pid/status"
rmdir /sys/fs/cgroup/new/proxied/inner /sys/fs/cgroup/new/proxied /sys/fs/cgroup/new
mkdir -p /sys/fs/cgroup/new/proxied/recreated
kill -CONT "$manager_pid"
wait_until probe new/proxied/recreated tcp4 198.51.100.10:18080 origin proxy
mkdir /sys/fs/cgroup/new/proxied/recreated/later
wait_until probe new/proxied/recreated/later udp4 198.51.100.10:18080 origin proxy

rmdir /sys/fs/cgroup/new/proxied/recreated/later /sys/fs/cgroup/new/proxied/recreated \
	/sys/fs/cgroup/new/proxied /sys/fs/cgroup/new
if grep -q 'Failed to update cgroup routes' "$CASE_WORK/cgtproxy.log"; then
	echo 'Nested cgroup routes failed to reconcile' >&2
	exit 1
fi
