#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
# Duplicate, contained and adjacent ranges must form a union for both families.
# Traffic outside that union must still be proxied, locally and when forwarded.
for source in proxied forwarded; do
	for network in tcp4 udp4; do
		probe "$source" "$network" 198.51.100.20:18080 origin direct
		probe "$source" "$network" 198.51.100.10:18080 origin proxy
	done
	for network in tcp6 udp6; do
		probe "$source" "$network" '[2001:db8::20]:18080' origin direct
		probe "$source" "$network" '[2001:db8::10]:18080' origin proxy
	done
done
stop_cgtproxy

# Reuse the same routing configuration with empty and full-address-space sets.
sed '/^bypass:/,/^tproxies:/{ /^bypass:/b; /^tproxies:/b; d; }' "$CASE_DIR/config.yaml" >"$CASE_WORK/base.yaml"
CASE_DIR=$CASE_WORK
sed 's/^bypass:/bypass: []/' "$CASE_WORK/base.yaml" >"$CASE_DIR/config.yaml"
start_cgtproxy
probe proxied tcp4 198.51.100.20:18080 origin proxy
probe proxied tcp6 '[2001:db8::20]:18080' origin proxy
stop_cgtproxy
sed '/^bypass:/a\  - 0.0.0.0/0\n  - 255.255.255.255\n  - ::/0\n  - ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff' "$CASE_WORK/base.yaml" >"$CASE_DIR/config.yaml"
start_cgtproxy
for source in proxied forwarded; do
	for network in tcp4 udp4; do
		probe "$source" "$network" 198.51.100.10:18080 origin direct
	done
	for network in tcp6 udp6; do
		probe "$source" "$network" '[2001:db8::10]:18080' origin direct
	done
done
