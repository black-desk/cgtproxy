#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

mkdir /sys/fs/cgroup/proxied/gone
gone_id=$(stat -c %i /sys/fs/cgroup/proxied/gone)
start_cgtproxy
probe proxied tcp4 198.51.100.10:18080 origin proxy
probe proxied/gone tcp4 198.51.100.10:18080 origin direct

# Another connection must not be able to corrupt the owner's map. The former
# runtime-update-failure case relied on this edit succeeding to inject a fault.
nft list map inet cgtproxy cgroup-vmap >"$CASE_WORK/map-before.nft"
if LC_ALL=C nft flush map inet cgtproxy cgroup-vmap >"$CASE_WORK/external-edit.log" 2>&1; then
	echo 'External connection unexpectedly changed the owner table' >&2
	exit 1
fi
grep -q 'Operation not permitted' "$CASE_WORK/external-edit.log"
nft list map inet cgtproxy cgroup-vmap >"$CASE_WORK/map-after.nft"
diff -u "$CASE_WORK/map-before.nft" "$CASE_WORK/map-after.nft"

# The original lasting connection must still be able to reconcile removals
# and additions after the rejected external transaction.
rmdir /sys/fs/cgroup/proxied/gone
element_removed() {
	# Query the original ID: nft cannot resolve its path once the cgroup is gone.
	if LC_ALL=C nft get element inet cgtproxy cgroup-vmap "{ $gone_id }" >"$CASE_WORK/removed-element.log" 2>&1; then
		return 1
	fi
	grep -q 'No such file or directory' "$CASE_WORK/removed-element.log"
}
wait_until element_removed
mkdir /sys/fs/cgroup/proxied/gone
wait_until probe proxied/gone tcp4 198.51.100.10:18080 origin direct
probe proxied/gone udp6 '[2001:db8::10]:18080' origin direct
probe proxied tcp4 198.51.100.10:18080 origin proxy
kill -0 "$cgtproxy_pid"
