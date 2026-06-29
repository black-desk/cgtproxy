// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"net"

	. "github.com/black-desk/lib/go/ginkgo-helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

// These specs exercise the pure IP-arithmetic helpers. They neither require
// privileges nor depend on a cgroup root, so they are kept in a standalone
// Describe that the gated "Netfilter table" suite picks up automatically.

var _ = Describe("IP helpers", func() {
	var nft *NFTManager

	BeforeEach(func() {
		nft = &NFTManager{log: zap.NewNop().Sugar()}
	})

	Describe("nextIP", func() {
		ContextTable("nextIP(%s)",
			ContextTableEntry("1.2.3.4", "1.2.3.5").WithFmt("1.2.3.4"),
			ContextTableEntry("10.0.0.0", "10.0.0.1").WithFmt("10.0.0.0"),
			ContextTableEntry("1.2.3.255", "1.2.4.0").WithFmt("1.2.3.255"),
			ContextTableEntry("2001:db8::1", "2001:db8::2").WithFmt("2001:db8::1"),
			func(input, expected string) {
				It("should return the incremented address", func() {
					got := nft.nextIP(net.ParseIP(input))
					Expect(got.Equal(net.ParseIP(expected))).To(BeTrue(),
						"expected %s, got %s", expected, got)
				})
			})
	})

	Describe("lastIP", func() {
		ContextTable("lastIP(%s)",
			ContextTableEntry("192.168.1.0/24", "192.168.1.255").WithFmt("192.168.1.0/24"),
			ContextTableEntry("10.0.0.0/8", "10.255.255.255").WithFmt("10.0.0.0/8"),
			ContextTableEntry("172.16.0.0/12", "172.31.255.255").WithFmt("172.16.0.0/12"),
			ContextTableEntry("2001:db8::/32",
				"2001:db8:ffff:ffff:ffff:ffff:ffff:ffff").WithFmt("2001:db8::/32"),
			func(cidr, expected string) {
				It("should return the broadcast address of the range", func() {
					_, ipnet, err := net.ParseCIDR(cidr)
					Expect(err).ToNot(HaveOccurred())

					got := nft.lastIP(ipnet)
					Expect(got.Equal(net.ParseIP(expected))).To(BeTrue(),
						"expected %s, got %s", expected, got)
				})
			})
	})
})
