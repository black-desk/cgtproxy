#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

LOG_LEVEL=info "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
	--log-file "$CASE_WORK/check.log" check config
grep -Eq '"(level|L)":"(info|INFO)"' "$CASE_WORK/check.log"
[[ $(stat -c '%a' "$CASE_WORK/check.log") == 600 ]]
