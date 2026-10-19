#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

pids=()
server_number=0
cgtproxy_pid=

cleanup() {
	local status=$?
	trap - EXIT
	if ((status != 0)); then
		nft list ruleset >"$CASE_WORK/ruleset" 2>&1 || true
		ip rule show >"$CASE_WORK/ip-rules" 2>&1 || true
		ip route show table all >"$CASE_WORK/ip-routes" 2>&1 || true
		ip -6 neigh show >"$CASE_WORK/neighbors" 2>&1 || true
		for ns in "${client_ns:-}" "${origin_ns:-}"; do
			nsenter -t "$ns" -n ip -6 route show >>"$CASE_WORK/peer-routes" 2>&1 || true
			nsenter -t "$ns" -n ip -6 neigh show >>"$CASE_WORK/peer-routes" 2>&1 || true
		done
	fi
	if [[ -n $cgtproxy_pid ]]; then
		kill -TERM "$cgtproxy_pid" 2>/dev/null || true
		wait "$cgtproxy_pid" 2>/dev/null || true
	fi
	for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; done
	for pid in "${pids[@]}"; do wait "$pid" 2>/dev/null || true; done
	exit "$status"
}

in_group() {
	echo "$BASHPID" >"/sys/fs/cgroup/$1/cgroup.procs"
	shift
	exec "$@"
}

wait_until() {
	for ((attempt = 0; attempt < 100; attempt++)); do
		if "$@"; then return 0; fi
		sleep 0.05
	done
	echo "Timed out waiting for: $*" >&2
	return 1
}

namespace_ready() {
	[[ $(readlink /proc/self/ns/net) != "$(readlink /proc/"$1"/ns/net)" ]]
}

setup_network() {
	unshare -n sleep infinity &
	client_ns=$!
	pids+=("$client_ns")
	unshare -n sleep infinity &
	origin_ns=$!
	pids+=("$origin_ns")
	# Wait until unshare has actually entered the new network namespace.
	wait_until namespace_ready "$client_ns"
	wait_until namespace_ready "$origin_ns"
	ip link add docker0 type bridge
	ip address add 172.31.254.1/24 dev docker0
	ip -6 address add fd42::1/64 dev docker0 nodad
	ip link set docker0 up
	ip link add bridge-peer type veth peer name eth0 netns "$client_ns"
	ip link set bridge-peer master docker0
	ip link set bridge-peer up
	for ns in "$client_ns" "$origin_ns"; do nsenter -t "$ns" -n ip link set lo up; done
	nsenter -t "$client_ns" -n ip address add 172.31.254.2/24 dev eth0
	nsenter -t "$client_ns" -n ip -6 address add fd42::2/64 dev eth0 nodad
	nsenter -t "$client_ns" -n ip link set eth0 up
	nsenter -t "$client_ns" -n ip route add default via 172.31.254.1
	nsenter -t "$client_ns" -n ip -6 route add default via fd42::1

	ip link add wan type veth peer name eth0 netns "$origin_ns"
	ip address add 198.18.0.1/24 dev wan
	ip -6 address add fd43::1/64 dev wan nodad
	ip link set wan up
	nsenter -t "$origin_ns" -n ip address add 198.18.0.2/24 dev eth0
	nsenter -t "$origin_ns" -n ip -6 address add fd43::2/64 dev eth0 nodad
	nsenter -t "$origin_ns" -n ip link set eth0 up
	for address in 198.51.100.10/32 198.51.100.20/32; do
		nsenter -t "$origin_ns" -n ip address add "$address" dev lo
	done
	for address in 2001:db8::10/128 2001:db8::20/128; do
		nsenter -t "$origin_ns" -n ip -6 address add "$address" dev lo nodad
	done
	ip route add 198.51.100.0/24 via 198.18.0.2
	ip -6 route add 2001:db8::/64 via fd43::2
	nsenter -t "$origin_ns" -n ip route add default via 198.18.0.1
	nsenter -t "$origin_ns" -n ip -6 route add default via fd43::1
	sysctl -qw net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1
	# Reproduce Docker's raw ingress-interface guard as well as its bridge path.
	nft -f - <<'NFT'
table ip docker-test {
    chain prerouting {
        type filter hook prerouting priority raw; policy accept;
        ip daddr 172.31.254.2 iifname != "docker0" drop
    }
    chain postrouting {
        type nat hook postrouting priority srcnat; policy accept;
        ip saddr 172.31.254.0/24 oifname "wan" masquerade
    }
}
NFT
}

start_server() {
	local ns=$1 network=$2 address=$3 id=$4
	shift 4
	server_number=$((server_number + 1))
	local ready=$CASE_WORK/server-$server_number.ready
	local command=("$FIXTURE_BIN" serve -network "$network" -address "$address" -id "$id" -ready "$ready" "$@")
	if [[ $ns != host ]]; then command=(nsenter -t "$ns" -n "${command[@]}"); fi
	in_group servers "${command[@]}" >"$CASE_WORK/server-$server_number.receipts" 2>"$CASE_WORK/server-$server_number.log" &
	pids+=("$!")
	wait_until test -f "$ready"
}

start_servers() {
	for network in tcp4 udp4 tcp6 udp6; do
		local any=0.0.0.0
		if [[ $network == *6 ]]; then any='[::]'; fi
		start_server host "$network" "$any:17893" proxy -transparent
		start_server "$origin_ns" "$network" "$any:18080" origin
		start_server "$origin_ns" "$network" "$any:53" origin
	done
	start_server host tcp4 0.0.0.0:17894 second -transparent
	start_server host tcp4 0.0.0.0:15353 dns
	start_server host udp4 0.0.0.0:15353 dns
	start_server host tcp4 0.0.0.0:18081 host
}

output_ready() {
	kill -0 "$cgtproxy_pid" 2>/dev/null && nft list chain inet cgtproxy output-mangle 2>/dev/null | grep -q 'goto cgroup-root'
}

start_cgtproxy() {
	in_group manager "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" --reuse-netlink-socket="$REUSE_NETLINK" \
		>"$CASE_WORK/cgtproxy.log" 2>&1 &
	cgtproxy_pid=$!
	wait_until output_ready
}

stop_cgtproxy() {
	if [[ -z $cgtproxy_pid ]]; then return; fi
	kill -TERM "$cgtproxy_pid"
	wait "$cgtproxy_pid"
	cgtproxy_pid=
	if nft list table inet cgtproxy >/dev/null 2>&1; then
		echo 'cgtproxy left its nft table behind' >&2
		return 1
	fi
	if {
		ip rule show
		ip -6 rule show
	} | grep -q 'lookup 31300'; then
		echo 'cgtproxy left its policy rules behind' >&2
		return 1
	fi
	if [[ -n $(ip route show table 31300 2>/dev/null) || -n $(ip -6 route show table 31300 2>/dev/null) ]]; then
		echo 'cgtproxy left its policy routes behind' >&2
		return 1
	fi
}

probe() {
	local source=$1 network=$2 address=$3 expect=$4 before
	local command=("$FIXTURE_BIN" client -network "$network" -address "$address" -expect "$expect")
	if [[ $expect == proxy || $expect == second || -n ${5:-} ]]; then command+=(-destination "${5:-$address}"); fi
	if [[ $expect == drop ]]; then before=$(cat "$CASE_WORK"/*.receipts | wc -l); fi
	if [[ $source == forwarded ]]; then
		nsenter -t "$client_ns" -n "${command[@]}"
	else
		(in_group "$source" "${command[@]}")
	fi
	if [[ $expect == drop ]]; then
		[[ $before == "$(cat "$CASE_WORK"/*.receipts | wc -l)" ]]
	fi
}

has_cgroup() {
	nft list map inet cgtproxy cgroup-vmap | grep -Fq "\"$1\""
}

no_cgroup() { ! has_cgroup "$1"; }
