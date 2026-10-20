// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"fmt"
	"math"

	"github.com/google/nftables"
	"golang.org/x/sys/unix"
)

// TODO: Contribute automatic SetAddElements/SetDeleteElements splitting to
// github.com/google/nftables, and length overflow rejection to
// github.com/mdlayher/netlink. Remove this workaround once upstream supports it.
//
// queueCgroupElements follows nft's mnl_nft_setelem_batch: an element that would
// overflow NFTA_SET_ELEM_LIST_ELEMENTS starts the next message. Only queue the
// messages here; the caller must Flush once to preserve transaction atomicity.
func queueCgroupElements(set *nftables.Set, elements []nftables.SetElement,
	update func(*nftables.Set, []nftables.SetElement) error,
) error {
	start, size := 0, unix.NLA_HDRLEN
	for i, element := range elements {
		elementSize := cgroupElementEncodedSize(element)
		if unix.NLA_HDRLEN+elementSize > math.MaxUint16 {
			return fmt.Errorf("cgroup element needs %d bytes, exceeding the netlink attribute limit", elementSize)
		}
		if size+elementSize > math.MaxUint16 {
			if err := update(set, elements[start:i]); err != nil {
				return err
			}
			start, size = i, unix.NLA_HDRLEN
		}
		size += elementSize
	}
	if start < len(elements) {
		return update(set, elements[start:])
	}
	return nil
}

// cgroupElementEncodedSize mirrors nftables v0.3.0's Set.makeElemList for the
// Key and VerdictData fields produced by genSetElement. Adding other fields to
// cgroup elements requires accounting for their encoding here as well.
func cgroupElementEncodedSize(element nftables.SetElement) int {
	key := alignedAttributeSize(alignedAttributeSize(len(element.Key)))
	verdict := alignedAttributeSize(4) // NFTA_VERDICT_CODE payload: 4-byte uint32
	if chain := element.VerdictData.Chain; chain != "" {
		verdict += alignedAttributeSize(len(chain) + 1) // NUL-terminated chain
	}
	data := alignedAttributeSize(alignedAttributeSize(verdict))
	return alignedAttributeSize(key + data)
}

func alignedAttributeSize(payload int) int {
	return (unix.NLA_HDRLEN + payload + unix.NLA_ALIGNTO - 1) & ^(unix.NLA_ALIGNTO - 1)
}
