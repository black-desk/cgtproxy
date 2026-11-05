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
	# Distinct sing-box outbound addresses let origins identify the proxy path.
	ip address add 198.18.0.3/24 dev wan
	ip address add 198.18.0.4/24 dev wan
	ip -6 address add fd43::1/64 dev wan nodad
	ip -6 address add fd43::3/64 dev wan nodad
	ip -6 address add fd43::4/64 dev wan nodad
	ip link set wan up
	nsenter -t "$origin_ns" -n -- bash -eu <<-'BASH'
		ip link set lo up
		ip address add 198.18.0.2/24 dev eth0
		ip -6 address add fd43::2/64 dev eth0 nodad
		ip link set eth0 up

		# Keep destination addresses separate from link addresses. Two
		# targets on the same server let cases test bypass and DNAT
		# between destinations.
		for address in 198.51.100.10/32 198.51.100.20/32 198.51.100.53/32; do
			ip address add "$address" dev lo
		done
		for address in 2001:db8::10/128 2001:db8::20/128; do
			ip -6 address add "$address" dev lo nodad
		done
		ip route add default via 198.18.0.1
		ip -6 route add default via fd43::1
	BASH
	# Keep automatic source selection stable despite the extra proxy IPs.
	ip route add 198.51.100.0/24 via 198.18.0.2 src 198.18.0.1
	ip -6 route add 2001:db8::/64 via fd43::2 src fd43::1
	# sing-box uses the default route to identify its external interface.
	# Both gateways remain inside this isolated topology.
	ip route add default via 198.18.0.2
	ip -6 route add default via fd43::2
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
		local addresses=(198.51.100.10 198.51.100.20)
		if [[ $network == *6 ]]; then addresses=('[2001:db8::10]' '[2001:db8::20]'); fi
		for address in "${addresses[@]}"; do
			start_server "$origin_ns" "$network" "$address:18080" origin
			start_server "$origin_ns" "$network" "$address:53" origin
		done
	done
	for address in 127.0.0.1 172.31.254.1; do
		start_server host tcp4 "$address:15353" dns
		start_server host udp4 "$address:15353" dns
	done
	# The external-DNAT case uses a remote DNS target: a local echo listener
	# would conflict with sing-box binding that same UDP port for write-back.
	start_server "$origin_ns" tcp4 198.51.100.53:15353 dns
	start_server "$origin_ns" udp4 198.51.100.53:15353 dns
	start_server host tcp4 172.31.254.1:18081 host

	# Proxy outbound sockets are exempt from interception through servers.
	in_group servers "$FIXTURE_BIN" proxy -config "$suite/sing-box.json" -ready "$CASE_WORK/proxy.ready" >"$CASE_WORK/sing-box.log" 2>&1 &
	sing_box_pid=$!
	pids+=("$sing_box_pid")
	if ! wait_until sing_box_ready; then
		cat "$CASE_WORK/sing-box.log" >&2
		return 1
	fi
}

sing_box_ready() {
	kill -0 "$sing_box_pid" 2>/dev/null && test -f "$CASE_WORK/proxy.ready"
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

# Arguments: cgroup/forwarded, transport, requested address, replying service,
# expected path (direct/proxy/second), optional translated destination and count.
# The translated destination defaults to the requested address; count defaults
# to two messages on one connection. The client checks the replying service,
# destination, payload, and source IP. Direct and proxied requests reach the
# same echo service, so success alone is insufficient: each proxy binds a
# distinct outbound IP, which the echo service reports to identify the path.
#
# Examples:
#   probe proxied tcp4 198.51.100.10:18080 origin proxy
#   probe direct udp4 198.51.100.10:18080 origin direct
#   probe forwarded udp4 198.51.100.10:53 dns direct 172.31.254.1:15353
probe() {
	local source=$1 network=$2 address=$3 server=$4 path=$5
	local destination=${6:-$address} source_ip=198.18.0.1
	if [[ $network == *6 ]]; then source_ip=fd43::1; fi
	case $path in
	proxy) source_ip=198.18.0.3; if [[ $network == *6 ]]; then source_ip=fd43::3; fi ;;
	second) source_ip=198.18.0.4; if [[ $network == *6 ]]; then source_ip=fd43::4; fi ;;
	direct)
		# IPv4 forwarding to the origin is masqueraded, but traffic to local
		# DNS/host services keeps the client's address. IPv6 is never NATed.
		if [[ $source == forwarded ]]; then
			if [[ $network == *6 ]]; then source_ip=fd42::2
			elif [[ $destination == 172.31.254.1:* || $destination == 127.0.0.1:* ]]; then source_ip=172.31.254.2; fi
		fi
		;;
	*) echo "Unknown expected path: $path" >&2; return 1 ;;
	esac
	local command=("$FIXTURE_BIN" client -network "$network" -address "$address" \
		-expect "$server" -source-ip "$source_ip" -destination "$destination" -count "${7:-2}")
	if [[ $source == forwarded ]]; then
		nsenter -t "$client_ns" -n "${command[@]}"
	else
		(in_group "$source" "${command[@]}")
	fi
}
