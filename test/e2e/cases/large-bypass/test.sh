#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Each family's interval boundaries exceed a 16-bit netlink attribute, while
# the complete transaction stays within the default netlink send buffer.
for ((i = 0; i < 2000; i++)); do
	printf '10.0.%d.%d\n' "$((2 * i / 256))" "$((2 * i % 256))"
done >"$CASE_WORK/ipv4"
for ((i = 0; i < 1700; i++)); do
	printf '2001:db8:1::%x\n' "$((2 * i))"
done >"$CASE_WORK/ipv6"
echo 198.51.100.20 >>"$CASE_WORK/ipv4"
echo 2001:db8::20 >>"$CASE_WORK/ipv6"
sed '/^bypass:/d' "$CASE_DIR/config.yaml" >"$CASE_WORK/config.yaml"
{
	echo 'bypass:'
	while read -r address; do printf '  - "%s"\n' "$address"; done <"$CASE_WORK/ipv4"
	while read -r address; do printf '  - "%s"\n' "$address"; done <"$CASE_WORK/ipv6"
} >>"$CASE_WORK/config.yaml"
CASE_DIR=$CASE_WORK

# Splitting messages must preserve the single initial transaction.
"$FIXTURE_BIN" check-nft-init -ready "$CASE_WORK/monitor.ready" -cgroup /sys/fs/cgroup/servers &
monitor_pid=$!
pids+=("$monitor_pid")
wait_until test -f "$CASE_WORK/monitor.ready"
start_cgtproxy
wait "$monitor_pid"
"$FIXTURE_BIN" check-bypass-set -set bypass -addresses "$CASE_WORK/ipv4"
"$FIXTURE_BIN" check-bypass-set -set bypass6 -addresses "$CASE_WORK/ipv6"

for source in proxied forwarded; do
	for network in tcp4 udp4; do
		probe "$source" "$network" 198.51.100.20:18080 origin
		probe "$source" "$network" 198.51.100.10:18080 proxy
	done
	for network in tcp6 udp6; do
		probe "$source" "$network" '[2001:db8::20]:18080' origin
		probe "$source" "$network" '[2001:db8::10]:18080' proxy
	done
done
