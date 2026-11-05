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
		# Wake a manager paused by a case before requesting graceful shutdown.
		kill -CONT "$cgtproxy_pid" 2>/dev/null || true
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
	# The current network namespace is the simulated host, running cgtproxy
	# and the transparent proxy. Local probes run here in different cgroups;
	# forwarded probes run in a separate client namespace.
	#
	# client eth0 -- veth -- docker0 [host] wan -- veth -- eth0 origin
	# 172.31.254.2           .1        198.18.0.1         .2
	# fd42::2               ::1       fd43::1            ::2
	#
	# The origin also owns the test destination addresses on its loopback.
	# All networks are private to this case; no Internet connection is needed.
	# These variables hold PIDs: sleep keeps each namespace alive so nsenter
	# can select it with -t, and cleanup can terminate the holder process.
	unshare -n sleep infinity &
	client_ns=$!
	pids+=("$client_ns")
	unshare -n sleep infinity &
	origin_ns=$!
	pids+=("$origin_ns")
	# Wait until unshare has actually entered the new network namespace.
	wait_until namespace_ready "$client_ns"
	wait_until namespace_ready "$origin_ns"

	# Client side: reproduce a container connected to a host bridge. Its
	# default gateway is the host, so requests exercise the forwarding path.
	# nodad skips IPv6 duplicate-address detection on these isolated links,
	# making each address usable immediately after setup.
	ip link add docker0 type bridge
	ip address add 172.31.254.1/24 dev docker0
	ip -6 address add fd42::1/64 dev docker0 nodad
	ip link set docker0 up
	ip link add bridge-peer type veth peer name eth0 netns "$client_ns"
	ip link set bridge-peer master docker0
	ip link set bridge-peer up
	nsenter -t "$client_ns" -n -- bash -eu <<-'BASH'
		ip link set lo up
		ip address add 172.31.254.2/24 dev eth0
		ip -6 address add fd42::2/64 dev eth0 nodad
		ip link set eth0 up
		ip route add default via 172.31.254.1
		ip -6 route add default via fd42::1
	BASH

	# Origin side: a separate routed link represents the external network.
	# Replies return through the host, including replies to forwarded probes.
	ip link add wan type veth peer name eth0 netns "$origin_ns"
	ip address add 198.18.0.1/24 dev wan
	ip -6 address add fd43::1/64 dev wan nodad
	ip link set wan up
	nsenter -t "$origin_ns" -n -- bash -eu <<-'BASH'
		ip link set lo up
		ip address add 198.18.0.2/24 dev eth0
		ip -6 address add fd43::2/64 dev eth0 nodad
		ip link set eth0 up

		# Keep destination addresses separate from link addresses. Two
		# targets on the same server let cases test bypass and DNAT
		# between destinations.
		for address in 198.51.100.10/32 198.51.100.20/32; do
			ip address add "$address" dev lo
		done
		for address in 2001:db8::10/128 2001:db8::20/128; do
			ip -6 address add "$address" dev lo nodad
		done
		ip route add default via 198.18.0.1
		ip -6 route add default via fd43::1
	BASH
	ip route add 198.51.100.0/24 via 198.18.0.2
	ip -6 route add 2001:db8::/64 via fd43::2
	sysctl -qw net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1

	# Reproduce Docker-style IPv4 rules without running Docker: the raw
	# guard drops packets to the client arriving outside docker0, while
	# masquerade translates client traffic leaving through wan to the host's
	# address. This exercises compatibility with those rules alongside
	# cgtproxy.
	nft -f - <<-'NFT'
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

rules_ready() {
	kill -0 "$cgtproxy_pid" 2>/dev/null && \
		nft list table inet cgtproxy >/dev/null 2>&1
}

start_cgtproxy() {
	LOG_LEVEL=${LOG_LEVEL:-debug} in_group manager \
		"$CGTPROXY_BIN" \
		--config "$CASE_DIR/config.yaml" \
		--log-file "$CASE_WORK/cgtproxy.log" &
	cgtproxy_pid=$!
	wait_until rules_ready
}

# SIGTERM lets cgtproxy remove its kernel rules/routes and write coverage
# before we inspect cleanup; namespace teardown alone would skip that work.
stop_cgtproxy() {
	if [[ -z $cgtproxy_pid ]]; then return; fi
	kill -TERM "$cgtproxy_pid"
	wait "$cgtproxy_pid"
	cgtproxy_pid=
	assert_network_cleanup
}

assert_network_cleanup() {
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
	local source=$1 network=$2 address=$3 expect=$4
	local command=("$FIXTURE_BIN" client -network "$network" -address "$address" -expect "$expect")
	if [[ $expect == proxy || $expect == second || -n ${5:-} ]]; then command+=(-destination "${5:-$address}"); fi
	if [[ $source == forwarded ]]; then
		nsenter -t "$client_ns" -n "${command[@]}"
	else
		(in_group "$source" "${command[@]}")
	fi
}
