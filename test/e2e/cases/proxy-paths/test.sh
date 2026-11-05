#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

mkdir /sys/fs/cgroup/proxied/second
start_cgtproxy
for network in tcp4 udp4 tcp6 udp6; do
	address=198.51.100.10:18080
	if [[ $network == *6 ]]; then address='[2001:db8::10]:18080'; fi
	probe proxied "$network" "$address" origin proxy "$address" 4
	probe proxied/second "$network" "$address" origin second "$address" 4
	probe direct "$network" "$address" origin direct "$address" 4
done

# A successful origin response is insufficient: reject a direct connection
# mistaken for proxying, and reject selection of the wrong proxy.
if probe direct tcp4 198.51.100.10:18080 origin proxy >"$CASE_WORK/wrong-source.log" 2>&1; then
	echo 'Direct traffic unexpectedly satisfied a proxy assertion' >&2
	exit 1
fi
grep -q 'unexpected source' "$CASE_WORK/wrong-source.log"
if probe proxied/second udp6 '[2001:db8::10]:18080' origin proxy >"$CASE_WORK/wrong-proxy.log" 2>&1; then
	echo 'Second proxy unexpectedly satisfied the first proxy assertion' >&2
	exit 1
fi
grep -q 'unexpected source' "$CASE_WORK/wrong-proxy.log"
