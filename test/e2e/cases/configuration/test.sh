#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

"$CGTPROXY_BIN" --version
"$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" check config
repo=$(cd -- "$CASE_DIR/../../../.." && pwd)
"$CGTPROXY_BIN" --config "$repo/misc/config/example.yaml" check config
"$CGTPROXY_BIN" --config "$repo/test/data/example_config.yaml" check config
check_invalid_config() {
	local invalid=$1 path=$2 status=0
	"$CGTPROXY_BIN" --config "$path" check config || status=$?
	if [[ $status != 1 ]]; then
		echo "Expected configuration error for $invalid, got status $status" >&2
		exit 1
	fi

	status=0
	timeout 5s "$CGTPROXY_BIN" --config "$path" \
		--log-file "$CASE_WORK/$invalid.log" || status=$?
	if [[ $status != 1 ]]; then
		echo "Expected startup configuration error for $invalid, got status $status" >&2
		exit 1
	fi
}

for invalid in wrong_type validation_fail undefined_proxy null_proxy invalid_regex duplicate_mark; do
	check_invalid_config "$invalid" "$repo/test/data/$invalid.yaml"
done

# Invalid table IDs must fail validation before any network state is changed.
for table in -1 0 253 254 255 4294967296; do
	path=$CASE_WORK/route-table-$table.yaml
	sed "s/^route-table:.*/route-table: $table/" "$CASE_DIR/config.yaml" >"$path"
	check_invalid_config "route-table-$table" "$path"
done
for table in 1 252 256 31300; do
	path=$CASE_WORK/route-table-$table.yaml
	sed "s/^route-table:.*/route-table: $table/" "$CASE_DIR/config.yaml" >"$path"
	"$CGTPROXY_BIN" --config "$path" check config
done
start_cgtproxy
probe forwarded tcp4 198.51.100.10:18080 origin
