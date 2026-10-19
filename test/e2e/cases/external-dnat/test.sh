#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Other software can DNAT non-DNS connections before TPROXY dispatch.
nft -f - <<'NFT'
table inet external-dnat {
    chain translate {
        ip daddr 198.51.100.10 meta l4proto { tcp, udp } th dport { 53, 18080 } dnat ip to 198.51.100.10:18082
        ip daddr 198.51.100.10 meta l4proto { tcp, udp } th dport 18083 dnat ip to 172.31.254.1:15353
        ip6 daddr 2001:db8::10 meta l4proto { tcp, udp } th dport { 53, 18080 } dnat ip6 to [2001:db8::10]:18082
    }
    chain output {
        type nat hook output priority dstnat - 1; policy accept;
        jump translate
    }
    chain prerouting {
        type nat hook prerouting priority dstnat - 1; policy accept;
        jump translate
    }
}
NFT

for network in tcp4 udp4 tcp6 udp6; do
	any=0.0.0.0
	if [[ $network == *6 ]]; then any='[::]'; fi
	# shellcheck disable=SC2154
	start_server "$origin_ns" "$network" "$any:18082" origin
done
start_cgtproxy
for network in tcp4 udp4 tcp6 udp6; do
	address=198.51.100.10
	if [[ $network == *6 ]]; then address='[2001:db8::10]'; fi
	# External DNS DNAT must not be mistaken for cgtproxy's own hijack either.
	for port in 18080 53; do
		# Each probe sends two requests on the same connection, including after DNAT.
		probe proxied "$network" "$address:$port" proxy "$address:18082"
		probe forwarded "$network" "$address:$port" proxy "$address:18082"
		probe direct "$network" "$address:$port" origin "$address:18082"
	done
done
# A non-DNS DNAT can even target the configured resolver; the original port matters.
for network in tcp4 udp4; do
	for source in proxied forwarded; do
		probe "$source" "$network" 198.51.100.10:18083 proxy 172.31.254.1:15353
	done
	probe direct "$network" 198.51.100.10:18083 dns 172.31.254.1:15353
done
# Keep cgtproxy's own DNS DNAT working alongside the external NAT chains.
for source in proxied forwarded; do
	probe "$source" tcp4 198.51.100.20:53 dns
	probe "$source" udp4 198.51.100.20:53 dns
done
