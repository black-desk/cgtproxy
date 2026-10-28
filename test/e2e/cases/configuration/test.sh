#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

"$CGTPROXY_BIN" --version
"$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" check config
repo=$(cd -- "$CASE_DIR/../../../.." && pwd)
"$CGTPROXY_BIN" --config "$repo/misc/config/example.yaml" check config
"$CGTPROXY_BIN" --config "$repo/test/data/example_config.yaml" check config

# Error annotations must preserve literal percent signs in paths.
status=0
"$CGTPROXY_BIN" --config "$CASE_WORK/missing%config.yaml" check config \
	>"$CASE_WORK/missing-config.stderr" 2>&1 || status=$?
[[ $status == 1 ]]
grep -Fq "read configuration from $CASE_WORK/missing%config.yaml" "$CASE_WORK/missing-config.stderr"
if grep -Fq '%!' "$CASE_WORK/missing-config.stderr"; then
	echo 'Error annotation contains a formatting error' >&2
	exit 1
fi

# Profile setup must report template and file errors before starting the core.
for profile in cpu block; do
	for path in '{{' '{{call .PID}}' "$CASE_WORK"; do
		status=0
		CGTPROXY_PROFILE=$profile timeout 5s "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
			--log-file "$CASE_WORK/profile.log" "--$profile-profile" "$path" \
			>"$CASE_WORK/profile.stderr" 2>&1 || status=$?
		[[ $status == 1 ]]
		if [[ $path == "$CASE_WORK" ]]; then
			grep -Fq 'is a directory' "$CASE_WORK/profile.stderr"
		else
			grep -Fq 'template:' "$CASE_WORK/profile.stderr"
		fi
	done
done

check_invalid_config() {
	local invalid=$1 path=$2 status=0
	"$CGTPROXY_BIN" --config "$path" check config || status=$?
	if [[ $status != 1 ]]; then
		echo "Expected configuration error for $invalid, got status $status" >&2
		exit 1
	fi

	status=0
	timeout 5s "$CGTPROXY_BIN" --config "$path" \
		--log-file "$CASE_WORK/$invalid.log" >"$CASE_WORK/$invalid.stderr" 2>&1 || status=$?
	if [[ $status != 1 ]]; then
		cat "$CASE_WORK/$invalid.stderr" >&2
		echo "Expected startup configuration error for $invalid, got status $status" >&2
		exit 1
	fi
}

for invalid in wrong_type validation_fail undefined_proxy null_proxy invalid_regex duplicate_mark; do
	check_invalid_config "$invalid" "$repo/test/data/$invalid.yaml"
done

# Reject the removed target even when another valid target is present.
for value in true false; do
	path=$CASE_WORK/removed-drop-$value.yaml
	cat >"$path" <<YAML
version: 1
cgroup-root: AUTO
route-table: 31300
rules:
  - match: "/.*"
    direct: true
    drop: $value
YAML
	check_invalid_config "removed-drop-$value" "$path"
	grep -q 'field drop not found' "$CASE_WORK/removed-drop-$value.stderr"
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
# Proxy names share a namespace with all generated and internal chains.
for name in cgroup-root output-mangle output-nat output-bypass prerouting forward-mangle prerouting-nat; do
	path=$CASE_WORK/proxy-name-$name.yaml
	cat "$CASE_DIR/config.yaml" >"$path"
	cat >>"$path" <<YAML
tproxies:
  $name:
    port: 17893
    mark: 42042
YAML
	check_invalid_config "proxy-name-$name" "$path"
done
for suffix in MARK DNS; do
	path=$CASE_WORK/proxy-collision-$suffix.yaml
	cat "$CASE_DIR/config.yaml" >"$path"
	cat >>"$path" <<YAML
tproxies:
  proxy:
    port: 17893
    mark: 42042
    dns-hijack:
      port: 15353
  proxy-$suffix:
    port: 17894
    mark: 42043
YAML
	check_invalid_config "proxy-collision-$suffix" "$path"
done

# The suffixes themselves are valid when no generated names collide.
for name in proxy-MARK proxy-DNS; do
	path=$CASE_WORK/proxy-valid-$name.yaml
	cat "$CASE_DIR/config.yaml" >"$path"
	cat >>"$path" <<YAML
tproxies:
  $name:
    port: 17893
    mark: 42042
YAML
	"$CGTPROXY_BIN" --config "$path" check config
done
# A configured DNS hijack requires an explicit, nonzero destination port.
for port in missing 0 65536; do
	path=$CASE_WORK/dns-port-$port.yaml
	cat "$CASE_DIR/config.yaml" >"$path"
	cat >>"$path" <<'YAML'
tproxies:
  proxy:
    port: 17893
    mark: 42042
    dns-hijack:
      ip: 127.0.0.1
YAML
	if [[ $port != missing ]]; then printf '      port: %s\n' "$port" >>"$path"; fi
	check_invalid_config "dns-port-$port" "$path"
done
for port in 53 65535; do
	path=$CASE_WORK/dns-port-$port.yaml
	cat "$CASE_DIR/config.yaml" >"$path"
	cat >>"$path" <<YAML
tproxies:
  proxy:
    port: 17893
    mark: 42042
    dns-hijack:
      port: $port
YAML
	"$CGTPROXY_BIN" --config "$path" check config
done
start_cgtproxy
probe forwarded tcp4 198.51.100.10:18080 origin
