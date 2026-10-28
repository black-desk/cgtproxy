#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

for log_path in "" "$CASE_WORK/invalid-level.log"; do
	status=0
	LOG_LEVEL=invalid "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
		--log-file "$log_path" check config --with-logger >"$CASE_WORK/invalid-level.stderr" 2>&1 || status=$?
	[[ $status == 1 ]]
	grep -q 'invalid LOG_LEVEL' "$CASE_WORK/invalid-level.stderr"
done
