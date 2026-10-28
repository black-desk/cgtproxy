#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

LOG_LEVEL=error "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
	--log-file "$CASE_WORK/error-only.log" check config --with-logger
[[ ! -s $CASE_WORK/error-only.log ]]
