#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# File logging uses the build's encoder and warning stacktrace settings.
LOG_LEVEL= "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
	--log-file "$CASE_WORK/default-check.log" check config --with-logger
if go version -m "$CGTPROXY_BIN" | grep -q -- '-tags=.*debug'; then
	grep -q '"L":"WARN"' "$CASE_WORK/default-check.log"
	grep -q '"S":' "$CASE_WORK/default-check.log"
else
	grep -q '"level":"warn"' "$CASE_WORK/default-check.log"
	if grep -q '"stacktrace":' "$CASE_WORK/default-check.log"; then
		echo 'Regular builds must not attach stacktraces to warnings' >&2
		exit 1
	fi
fi
