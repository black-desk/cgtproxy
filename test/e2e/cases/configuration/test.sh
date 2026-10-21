#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

"$CGTPROXY_BIN" --version
"$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" check config
repo=$(cd -- "$CASE_DIR/../../../.." && pwd)
"$CGTPROXY_BIN" --config "$repo/misc/config/example.yaml" check config
"$CGTPROXY_BIN" --config "$repo/test/data/example_config.yaml" check config
for invalid in wrong_type validation_fail undefined_proxy null_proxy invalid_regex duplicate_mark; do
	status=0
	"$CGTPROXY_BIN" --config "$repo/test/data/$invalid.yaml" check config || status=$?
	if [[ $status != 1 ]]; then
		echo "Expected configuration error for $invalid, got status $status" >&2
		exit 1
	fi

	status=0
	timeout 5s "$CGTPROXY_BIN" --config "$repo/test/data/$invalid.yaml" \
		--log-file "$CASE_WORK/$invalid.log" || status=$?
	if [[ $status != 1 ]]; then
		echo "Expected startup configuration error for $invalid, got status $status" >&2
		exit 1
	fi
done
start_cgtproxy
probe forwarded tcp4 198.51.100.10:18080 origin
