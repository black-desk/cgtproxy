#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

"$FIXTURE_BIN" check-nft-init -ready "$CASE_WORK/monitor.ready" &
monitor_pid=$!
pids+=("$monitor_pid")
wait_until test -f "$CASE_WORK/monitor.ready"
start_cgtproxy
wait "$monitor_pid"
probe forwarded tcp4 198.51.100.10:18080 origin
probe forwarded tcp6 '[2001:db8::10]:18080' origin
probe forwarded udp6 '[2001:db8::10]:18080' origin
probe unmatched tcp4 198.51.100.10:18080 origin
