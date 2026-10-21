#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

mkdir /sys/fs/cgroup/second
start_cgtproxy
probe proxied tcp4 198.51.100.10:18080 proxy
probe second tcp4 198.51.100.10:18080 proxy
probe forwarded tcp4 198.51.100.10:18080 proxy
probe proxied tcp4 198.51.100.20:18080 origin

nft -f - <<'NFT'
table inet external-dnat {
    chain translate {
        ip daddr 198.51.100.10 meta l4proto { tcp, udp } th dport 18080 dnat ip to 198.51.100.20:18080
        ip6 daddr 2001:db8::10 meta l4proto { tcp, udp } th dport 18080 dnat ip6 to [2001:db8::20]:18080
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

# Each probe exchanges two messages, covering packets after conntrack's DNAT.
for source in direct proxied second forwarded; do
	for network in tcp4 udp4 tcp6 udp6; do
		original=198.51.100.10:18080
		translated=198.51.100.20:18080
		if [[ $network == *6 ]]; then
			original='[2001:db8::10]:18080'
			translated='[2001:db8::20]:18080'
		fi
		probe "$source" "$network" "$original" origin "$translated"
	done
done

# Bypassing must preserve marks owned by another network component.
nft -f - <<'NFT'
table inet foreign-mark {
    chain output {
        type route hook output priority mangle - 1; policy accept;
        ip daddr 198.51.100.20 meta mark set 777
        ip6 daddr 2001:db8::20 meta mark set 777
    }
    chain postrouting {
        type filter hook postrouting priority filter; policy accept;
        ip daddr 198.51.100.20 meta mark != 777 drop
        ip6 daddr 2001:db8::20 meta mark != 777 drop
    }
}
NFT
for network in tcp4 udp4 tcp6 udp6; do
	address=198.51.100.20:18080
	if [[ $network == *6 ]]; then address='[2001:db8::20]:18080'; fi
	probe direct "$network" "$address" origin
done
