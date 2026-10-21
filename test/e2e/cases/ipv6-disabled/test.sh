#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Fixtures are already running; change only the isolated host's IPv6 setting.
original_config=$CASE_DIR/config.yaml
sysctl -qw net.ipv6.conf.all.disable_ipv6=1
start_cgtproxy
for network in tcp4 udp4; do
	probe proxied "$network" 198.51.100.10:18080 proxy
	probe forwarded "$network" 198.51.100.10:18080 proxy
	probe direct "$network" 198.51.100.10:18080 origin
done
[[ -z $(ip -6 route show table 31300 2>/dev/null) ]]
if ip -6 rule show | grep -q 'lookup 31300'; then
	echo 'Installed an IPv6 policy rule on an IPv6-disabled host' >&2
	exit 1
fi
stop_cgtproxy

# An IPv4-only host must also work without any TPROXY targets.
cat >"$CASE_WORK/config.yaml" <<'YAML'
version: 1
cgroup-root: AUTO
route-table: 31300
rules:
  - match: "/.*"
    direct: true
YAML
export CASE_DIR=$CASE_WORK
start_cgtproxy
probe proxied tcp4 198.51.100.10:18080 origin
stop_cgtproxy

# A target explicitly allowing IPv6 still requires successful IPv6 setup.
sed '/^    no-ipv6:/d' "$original_config" >"$CASE_WORK/config.yaml"
status=0
(in_group manager timeout 5s "$CGTPROXY_BIN" --config "$CASE_WORK/config.yaml" \
	--log-file "$CASE_WORK/ipv6-required.log") || status=$?
if [[ $status != 1 ]]; then
	echo "Expected IPv6 initialization to fail, got status $status" >&2
	exit 1
fi
