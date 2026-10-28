#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Leave LOG_LEVEL empty to verify the build's default file logging level.
LOG_LEVEL= in_group manager "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
	--log-file "$CASE_WORK/cgtproxy.log" &
cgtproxy_pid=$!
wait_until output_ready
# Graceful shutdown also exercises the command's own debug logging.
stop_cgtproxy
if go version -m "$CGTPROXY_BIN" | grep -q -- '-tags=.*debug'; then
	grep -q '"L":"DEBUG"' "$CASE_WORK/cgtproxy.log"
else
	grep -q '"level":"info"' "$CASE_WORK/cgtproxy.log"
	if grep -q '"level":"debug"' "$CASE_WORK/cgtproxy.log"; then
		echo 'Regular builds must default to info logging' >&2
		exit 1
	fi
fi
