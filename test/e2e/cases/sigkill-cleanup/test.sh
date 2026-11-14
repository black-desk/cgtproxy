#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
for source in proxied forwarded; do
	for network in tcp4 udp4 tcp6 udp6; do
		address=198.51.100.10:18080
		if [[ $network == *6 ]]; then address='[2001:db8::10]:18080'; fi
		probe "$source" "$network" "$address" origin proxy
	done
	probe "$source" udp4 198.51.100.10:53 dns direct 172.31.254.1:15353
done
nft list table ip docker-test >"$CASE_WORK/foreign-before.nft"

# No graceful shutdown or namespace teardown may remove the table for us.
kill -KILL "$cgtproxy_pid"
status=0
wait "$cgtproxy_pid" || status=$?
cgtproxy_pid=
if [[ $status != 137 ]]; then
	echo "Expected SIGKILL exit status 137, got $status" >&2
	exit 1
fi

table_removed() {
	nft list ruleset >"$CASE_WORK/ruleset-after-kill.nft" || return 1
	! grep -q '^table inet cgtproxy {' "$CASE_WORK/ruleset-after-kill.nft"
}
wait_until table_removed
nft list table ip docker-test >"$CASE_WORK/foreign-after.nft"
diff -u "$CASE_WORK/foreign-before.nft" "$CASE_WORK/foreign-after.nft"

# SIGKILL can leave policy routes behind. New unmarked connections must still
# work directly once the owner table and its TPROXY/DNS rules are gone.
for source in proxied forwarded; do
	for network in tcp4 udp4 tcp6 udp6; do
		address=198.51.100.10:18080
		if [[ $network == *6 ]]; then address='[2001:db8::10]:18080'; fi
		probe "$source" "$network" "$address" origin direct
	done
	probe "$source" udp4 198.51.100.10:53 origin direct
done
