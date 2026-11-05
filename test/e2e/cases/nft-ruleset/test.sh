#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# This is the only case asserting nft rule contents. Compare text read from the
# application's table with a reviewed snapshot; never install a reference table.
# When rules change, inspect actual.raw.nft and comparison.diff, review the
# rule semantics, and deliberately update the base expected.nft snapshot.
# Do not blindly accept captured output. nft version changes can alter the
# displayed text too, so their differences also need review.
snapshot=$CASE_DIR/expected.nft
normalizer=$CASE_DIR/normalize.awk
base_config=$CASE_DIR/config.yaml
ranges_config=$CASE_DIR/ranges.yaml
root_id=$(stat -c %i /sys/fs/cgroup)
primary=proxy
gateway_rule='goto proxy-MARK'
deep_rule=
debug=0
case ",${E2E_GO_TAGS:-}," in *,debug,*) debug=1 ;; esac
extra_cgroups=$CASE_WORK/cgroups.elements
extra_bypass=$CASE_WORK/bypass.elements
extra_bypass6=$CASE_WORK/bypass6.elements
: >"$extra_cgroups"
: >"$extra_bypass"
: >"$extra_bypass6"
override_bypass=0
mkdir /sys/fs/cgroup/proxied/second

normalize() {
	LC_ALL=C awk -v root="$root_id" -f "$normalizer"
}

# Derive scenario expectations from the base snapshot: change only the gateway,
# lookup depth, proxy name, and explicitly enumerated set/map elements below.
build_expected() {
	awk -v primary="$primary" -v gateway="$gateway_rule" -v deep="$deep_rule" \
		-v debug="$debug" -v override="$override_bypass" \
		-v cgroups="$extra_cgroups" -v bypass="$extra_bypass" -v bypass6="$extra_bypass6" '
		function add(value) {
			sub(/^[\t ]+/, "", value)
			sub(/,?$/, "", value)
			elements = elements (elements == "" ? "" : ", ") value
		}
		/^[\t ]*(set|map|chain) / { kind = $1; object = $2 }
		/^[\t ]*elements = \{/ { in_elements = 1; elements = ""; next }
		in_elements {
			if ($0 !~ /^[\t ]*}/) {
				if (!(override && (object == "bypass" || object == "bypass6"))) add($0)
				next
			}
			file = (object == "cgroup-vmap" ? cgroups : object == "bypass" ? bypass : object == "bypass6" ? bypass6 : "")
			if (file != "") {
				while ((getline value < file) > 0) add(value)
				close(file)
			}
			$0 = "\t\telements = { " elements " }"
			in_elements = 0
		}
		/^\t\tgoto proxy-MARK$/ { if (gateway == "") next; $0 = "\t\t" gateway }
		/^\t\tsocket cgroupv2 level 2 / {
			if (deep != "") print "\t\t" (debug ? "counter packets 0 bytes 0 " : "") deep
		}
		{
			gsub(/proxy-MARK/, primary "-MARK")
			gsub(/proxy-DNS/, primary "-DNS")
			sub(/chain proxy \{/, "chain " primary " {")
			gsub(/goto proxy,/, "goto " primary ",")
			sub(/goto proxy$/, "goto " primary)
			# DNS DNAT chains currently have no debug counters. Every other
			# rule prepends one; retain its presence and position in the diff.
			if (debug && kind == "chain" && object !~ /-DNS$/ && $0 ~ /^\t\t[^\t}]/ && $0 !~ /^\t\ttype /)
				sub(/^\t\t/, "\t\tcounter packets 0 bytes 0 ")
			print
		}' "$snapshot" | normalize >"$CASE_WORK/expected.nft"
}

rules_match() {
	nft list table inet cgtproxy >"$CASE_WORK/actual.raw.nft" || return 1
	normalize <"$CASE_WORK/actual.raw.nft" >"$CASE_WORK/actual.nft" || return 1
	diff -u "$CASE_WORK/expected.nft" "$CASE_WORK/actual.nft" >"$CASE_WORK/comparison.diff"
}

assert_rules() {
	if ! wait_until rules_match; then
		cat "$CASE_WORK/comparison.diff" >&2
		return 1
	fi
}

start_cgtproxy
build_expected
assert_rules

# Adding the deepest cgroup must update both the map and descending socket
# lookup levels. Removing it must remove both, without disturbing other rules.
mkdir /sys/fs/cgroup/proxied/second/deep
deep_rule='socket cgroupv2 level 3 vmap @cgroup-vmap'
echo '"proxied/second/deep" : goto proxy-MARK' >"$extra_cgroups"
build_expected
assert_rules
rmdir /sys/fs/cgroup/proxied/second/deep
deep_rule=
: >"$extra_cgroups"
build_expected
assert_rules
stop_cgtproxy

# Gateway dispatch must follow its own target, including an explicit return
# and an omitted target. The local root verdict remains direct in every phase.
cp "$base_config" "$CASE_WORK/base.yaml"
CASE_DIR=$CASE_WORK
for gateway in direct omitted; do
	sed '/^gateway:/,/^cgroup-root:/{ /^cgroup-root:/!d; }' "$CASE_WORK/base.yaml" >"$CASE_DIR/config.yaml"
	gateway_rule=
	if [[ $gateway == direct ]]; then
		printf '\ngateway:\n  direct: true\n' >>"$CASE_DIR/config.yaml"
		gateway_rule=return
	fi
	start_cgtproxy
	build_expected
	assert_rules
	stop_cgtproxy
done

# Compare the full union of duplicate/overlapping/adjacent ranges, including
# mapped IPv4 and endpoints at zero and the maximum address in each family.
sed '/^bypass:/,/^tproxies:/{ /^tproxies:/!d; }' "$CASE_WORK/base.yaml" >"$CASE_DIR/config.yaml"
cat "$ranges_config" >>"$CASE_DIR/config.yaml"
gateway_rule='goto proxy-MARK'
override_bypass=1
printf '%s\n' 0.0.0.0/31 198.51.100.20/30 255.255.255.254/31 >"$extra_bypass"
printf '%s\n' ::/80 2001:db8::20/126 ffff:ffff:ffff:ffff:ffff:ffff:ffff:fffe/127 >"$extra_bypass6"
start_cgtproxy
build_expected
assert_rules
stop_cgtproxy
override_bypass=0
: >"$extra_bypass"
: >"$extra_bypass6"

# Check every large-map element using nft, including variable-length verdicts.
# Keep this separate from the large-set phase to stay below the netlink socket's
# transaction send-buffer limit while exceeding the per-attribute length limit.
primary=proxy-with-a-long-chain-name-to-exercise-variable-length-verdict-encoding-and-netlink-attribute-alignment-when-splitting-large-cgroup-maps-without-changing-the-transaction-boundaries-or-dropping-elements
gateway_rule="goto $primary-MARK"
for ((i = 0; i < 300; i++)); do mkdir "/sys/fs/cgroup/group-proxy-$i"; done
for ((i = 0; i < 20; i++)); do mkdir "/sys/fs/cgroup/group-direct-$i"; done
sed -e "s/^  proxy:/  $primary:/" -e "s/tproxy: proxy$/tproxy: $primary/" \
	"$CASE_WORK/base.yaml" >"$CASE_DIR/config.yaml"
cat >>"$CASE_DIR/config.yaml" <<YAML
  - match: "^/group-proxy-"
    tproxy: $primary
  - match: "^/group-direct-"
    direct: true
YAML
{
	for ((i = 0; i < 300; i++)); do printf '"group-proxy-%s" : goto %s-MARK\n' "$i" "$primary"; done
	for ((i = 0; i < 20; i++)); do printf '"group-direct-%s" : return\n' "$i"; done
} >"$extra_cgroups"
start_cgtproxy
build_expected
assert_rules
rmdir /sys/fs/cgroup/group-direct-0
sed -i '/"group-direct-0"/d' "$extra_cgroups"
build_expected
assert_rules
mkdir /sys/fs/cgroup/group-direct-0
echo '"group-direct-0" : return' >>"$extra_cgroups"
build_expected
assert_rules
stop_cgtproxy
for ((i = 0; i < 300; i++)); do rmdir "/sys/fs/cgroup/group-proxy-$i"; done
for ((i = 0; i < 20; i++)); do rmdir "/sys/fs/cgroup/group-direct-$i"; done

# Thousands of nonadjacent addresses require split interval-element messages.
# Compare every returned address against the independently enumerated inputs.
primary=proxy
gateway_rule='goto proxy-MARK'
for ((i = 0; i < 2000; i++)); do printf '10.0.%d.%d\n' "$((2 * i / 256))" "$((2 * i % 256))"; done >"$CASE_WORK/ipv4"
{
	echo 2001:db8:1::
	for ((i = 1; i < 1700; i++)); do printf '2001:db8:1::%x\n' "$((2 * i))"; done
} >"$CASE_WORK/ipv6"
{
	while read -r address; do printf '  - "%s"\n' "$address"; done <"$CASE_WORK/ipv4"
	while read -r address; do printf '  - "%s"\n' "$address"; done <"$CASE_WORK/ipv6"
} >"$CASE_WORK/extra-bypass.yaml"
awk -v entries="$CASE_WORK/extra-bypass.yaml" '/^tproxies:/ { while ((getline line < entries) > 0) print line } { print }' \
	"$CASE_WORK/base.yaml" >"$CASE_DIR/config.yaml"
: >"$extra_cgroups"
cp "$CASE_WORK/ipv4" "$extra_bypass"
cp "$CASE_WORK/ipv6" "$extra_bypass6"
start_cgtproxy
build_expected
assert_rules
stop_cgtproxy

# An unsuccessful startup must leave an existing same-name table unchanged.
# This content assertion belongs here; initialization-failure separately checks
# the public error, policy routing cleanup, and subsequent direct connectivity.
nft -f - <<'NFT'
table inet cgtproxy {
    chain sentinel {
        counter accept
    }
}
NFT
nft list table inet cgtproxy | normalize >"$CASE_WORK/sentinel-before.nft"
if (in_group manager "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" \
	--log-file "$CASE_WORK/collision.log"); then
	echo 'Startup unexpectedly replaced an existing table' >&2
	exit 1
fi
nft list table inet cgtproxy | normalize >"$CASE_WORK/sentinel-after.nft"
diff -u "$CASE_WORK/sentinel-before.nft" "$CASE_WORK/sentinel-after.nft"
nft delete table inet cgtproxy
assert_network_cleanup
