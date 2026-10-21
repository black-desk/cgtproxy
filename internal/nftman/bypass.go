// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"net/netip"
	"slices"

	"github.com/google/nftables"
)

// bypassSetElements follows nft's src/intervals.c: convert addresses and
// prefixes to closed ranges, sort and merge, then encode interval boundaries.
// All entries must already be validated and belong to the same address family.
func bypassSetElements(entries []string) []nftables.SetElement {
	type ipRange struct{ low, high netip.Addr }
	ranges := make([]ipRange, 0, len(entries))
	for _, entry := range entries {
		if addr, err := netip.ParseAddr(entry); err == nil {
			addr = addr.Unmap()
			ranges = append(ranges, ipRange{addr, addr})
			continue
		}
		prefix := netip.MustParsePrefix(entry).Masked()
		low := prefix.Addr()
		high := slices.Clone(low.AsSlice())
		for bit := prefix.Bits(); bit < low.BitLen(); bit++ {
			high[bit/8] |= 1 << (7 - bit%8)
		}
		last, _ := netip.AddrFromSlice(high)
		// Match WithBypass: mapped IPv4 prefixes belong to the IPv4 set.
		// A native IPv6 range may merely end inside the mapped region.
		if low.Is4In6() {
			low, last = low.Unmap(), last.Unmap()
		}
		ranges = append(ranges, ipRange{low, last})
	}
	slices.SortFunc(ranges, func(a, b ipRange) int {
		return a.low.Compare(b.low)
	})
	merged := ranges[:0]
	for _, current := range ranges {
		if len(merged) > 0 {
			prev := &merged[len(merged)-1]
			if current.low.Compare(prev.high) <= 0 || current.low == prev.high.Next() {
				if current.high.Compare(prev.high) > 0 {
					prev.high = current.high
				}
				continue
			}
		}
		merged = append(merged, current)
	}

	elements := make([]nftables.SetElement, 0, 2*len(merged)+1)
	for i, r := range merged {
		if i == 0 {
			zero := make([]byte, len(r.low.AsSlice()))
			addr, _ := netip.AddrFromSlice(zero)
			if r.low != addr {
				elements = append(elements, nftables.SetElement{Key: zero, IntervalEnd: true})
			}
		}
		elements = append(elements, nftables.SetElement{Key: r.low.AsSlice()})
		// A range ending at the maximum address has no end boundary. Next
		// returns an invalid address instead of wrapping around to zero.
		if end := r.high.Next(); end.IsValid() {
			elements = append(elements, nftables.SetElement{Key: end.AsSlice(), IntervalEnd: true})
		}
	}
	return elements
}
