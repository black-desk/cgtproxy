#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Match the shipped service's lack of DAC override/read-search capabilities.
export CGTPROXY_REAL_BIN=$CGTPROXY_BIN
cat >"$CASE_WORK/restricted-caps" <<'SH'
#!/bin/bash
exec setpriv --bounding-set=-dac_override,-dac_read_search "$CGTPROXY_REAL_BIN" "$@"
SH
chmod 700 "$CASE_WORK/restricted-caps"
CGTPROXY_BIN=$CASE_WORK/restricted-caps

# A failed initial scan must abort startup and leave no network rules behind.
mkdir /sys/fs/cgroup/unreadable
chmod 000 /sys/fs/cgroup/unreadable
status=0
(LOG_LEVEL=debug in_group manager "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
	--log-file "$CASE_WORK/initial-scan.log") || status=$?
chmod 755 /sys/fs/cgroup/unreadable
rmdir /sys/fs/cgroup/unreadable
if [[ $status != 1 ]]; then
	echo "Expected initial scan permission failure, got exit status $status" >&2
	exit 1
fi
grep -q 'running filesystem watcher: watch cgroup.*\/sys\/fs\/cgroup\/unreadable.*permission denied' "$CASE_WORK/initial-scan.log"
assert_network_cleanup

start_cgtproxy
probe proxied tcp4 198.51.100.10:18080 proxy

# Pause until the inaccessible directory is fully configured, so watching it
# fails rather than a later directory scan racing with chmod.
kill -STOP "$cgtproxy_pid"
wait_until grep -Eq '^State:[[:space:]]+T' "/proc/$cgtproxy_pid/status"
mkdir /sys/fs/cgroup/unreadable
chmod 000 /sys/fs/cgroup/unreadable
kill -CONT "$cgtproxy_pid"
daemon_exited() { ! kill -0 "$cgtproxy_pid" 2>/dev/null; }
wait_until daemon_exited
status=0
wait "$cgtproxy_pid" || status=$?
cgtproxy_pid=
chmod 755 /sys/fs/cgroup/unreadable
if [[ $status != 1 ]]; then
	echo "Expected watcher permission failure, got exit status $status" >&2
	exit 1
fi
grep -q 'watch cgroup.*\/sys\/fs\/cgroup\/unreadable.*permission denied' "$CASE_WORK/cgtproxy.log"
assert_network_cleanup
probe proxied tcp4 198.51.100.10:18080 origin
