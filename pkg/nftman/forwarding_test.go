// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package nftman

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

var _ = Describe("Root cgroup routing", func() {
	var nft *NFTManager
	BeforeEach(func() {
		if !(os.Geteuid() == 0 && os.Getenv("CGTPROXY_TEST_NFTMAN") == "1") {
			Skip(needSandboxMessage)
		}
		var err error
		nft, err = injectedNFTManagerWithLastingConnector(config.CGroupRoot(os.Getenv("CGTPROXY_TEST_CGROUP_ROOT")))
		Expect(err).To(Succeed())
		Expect(nft.InitStructure()).To(Succeed())
	})
	AfterEach(func() {
		if nft != nil {
			Expect(nft.Clear()).To(Succeed())
			Expect(nft.Release()).To(Succeed())
		}
	})

	It("shares the root chain and excludes local, reply and non-unicast traffic", func() {
		rules := getNFTableRules()
		Expect(rules).To(ContainSubstring("chain cgroup-root"))
		Expect(rules).To(MatchRegexp(`hook prerouting priority (mangle - 2|-152)`))
		Expect(rules).To(MatchRegexp(`hook prerouting priority (mangle - 1|-151)`))
		Expect(rules).To(ContainSubstring("fib daddr type local return"))
		Expect(rules).To(ContainSubstring("fib daddr type != unicast return"))
		Expect(rules).To(ContainSubstring("fib saddr type local return"))
		Expect(rules).To(ContainSubstring("ct direction reply return"))
		Expect(rules).To(ContainSubstring("goto cgroup-root"))
	})

	DescribeTable("updates the common root target",
		func(target types.Target, verdict expr.VerdictKind) {
			Expect(nft.SetRootTarget(types.Target{Op: types.TargetDrop})).To(Succeed())
			Expect(nft.SetRootTarget(target)).To(Succeed())
			conn, err := nft.connector.Connect()
			Expect(err).To(Succeed())
			rules, err := conn.GetRules(nft.table, nft.rootChain)
			Expect(err).To(Succeed())
			Expect(rules).To(HaveLen(1))
			Expect(rules[0].Exprs[len(rules[0].Exprs)-1]).To(Equal(&expr.Verdict{Kind: verdict}))
		},
		Entry("direct", types.Target{Op: types.TargetDirect}, expr.VerdictReturn),
		Entry("drop", types.Target{Op: types.TargetDrop}, expr.VerdictDrop),
		Entry("unmatched", types.Target{}, expr.VerdictReturn),
	)

	It("preserves explicit cgroup decisions before the root fallback", func() {
		root := os.Getenv("CGTPROXY_TEST_CGROUP_ROOT")
		path := root + "/explicit-direct"
		Expect(os.Mkdir(path, 0755)).To(Succeed())
		DeferCleanup(func() { Expect(os.Remove(path)).To(Succeed()) })
		Expect(nft.SetRootTarget(types.Target{Op: types.TargetDrop})).To(Succeed())
		Expect(nft.AddRoutes([]types.Route{{Path: path, Target: types.Target{Op: types.TargetDirect}}})).To(Succeed())
		rules := getNFTableRules()
		Expect(rules).To(MatchRegexp(`(?s)socket cgroupv2 level 1 vmap @cgroup-vmap.*goto cgroup-root`))
	})

	It("activates the local fallback after the first scan, even when no child rule matches", func() {
		Expect(nft.SetRootTarget(types.Target{Op: types.TargetDrop})).To(Succeed())
		conn, err := nft.connector.Connect()
		Expect(err).To(Succeed())
		hasFallback := func(rules []*nftables.Rule) bool {
			for _, rule := range rules {
				for _, expression := range rule.Exprs {
					if verdict, ok := expression.(*expr.Verdict); ok && verdict.Chain == nft.rootChain.Name {
						return true
					}
				}
			}
			return false
		}
		rules, err := conn.GetRules(nft.table, nft.outputMangleChain)
		Expect(err).To(Succeed())
		Expect(hasFallback(rules)).To(BeFalse())
		Expect(nft.AddRoutes(nil)).To(Succeed())
		rules, err = conn.GetRules(nft.table, nft.outputMangleChain)
		Expect(err).To(Succeed())
		Expect(hasFallback(rules)).To(BeTrue())
	})

	It("honors proxy protocol exclusions while allowing a configured DNS hijack", func() {
		tp := &config.TProxy{Name: "root-proxy", Port: 7893, Mark: 3000, NoIPv6: true, NoUDP: true,
			DNSHijack: &config.DNSHijack{IP: pstr("127.0.0.1"), Port: 5353}}
		Expect(nft.AddChainAndRulesForTProxies([]*config.TProxy{tp})).To(Succeed())
		Expect(nft.SetRootTarget(types.Target{Op: types.TargetTProxy, Chain: "root-proxy-MARK"})).To(Succeed())
		rules := getNFTableRules()
		Expect(rules).To(ContainSubstring("meta nfproto ipv6 return"))
		Expect(rules).To(ContainSubstring("udp dport != 53 return"))
		Expect(rules).To(ContainSubstring("goto root-proxy-MARK"))
		Expect(rules).To(ContainSubstring("meta mark set 0x00000000 dnat ip to 127.0.0.1:5353"))
	})

	DescribeTable("routes incoming TCP according to the root target and destination",
		func(ctx SpecContext, protocol, address, listenAddress string, rootOp types.TargetOp) {
			tp := &config.TProxy{Name: "forwarded", Port: 17893, Mark: 42042,
				DNSHijack: &config.DNSHijack{IP: pstr("172.31.254.1"), Port: 15353, TCP: true}}
			Expect(nft.AddChainAndRulesForTProxies([]*config.TProxy{tp})).To(Succeed())
			Expect(nft.SetRootTarget(types.Target{Op: rootOp, Chain: "forwarded-MARK"})).To(Succeed())
			lc := net.ListenConfig{Control: func(network, _ string, raw syscall.RawConn) error {
				var socketErr error
				err := raw.Control(func(fd uintptr) {
					if network == "tcp6" {
						socketErr = unix.SetsockoptInt(int(fd), unix.SOL_IPV6, unix.IPV6_TRANSPARENT, 1)
					} else {
						socketErr = unix.SetsockoptInt(int(fd), unix.SOL_IP, unix.IP_TRANSPARENT, 1)
					}
				})
				if err != nil {
					return err
				}
				return socketErr
			}}
			listener, err := lc.Listen(ctx, protocol, listenAddress)
			Expect(err).To(Succeed())
			DeferCleanup(listener.Close)
			received := make(chan string, 1)
			go func() {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				received <- conn.LocalAddr().String()
				_, _ = io.Copy(conn, io.LimitReader(conn, int64(len(forwardPayload))))
			}()
			done := startForwardClient(ctx, protocol, address)
			Eventually(received).Should(Receive(Equal(address)))
			Eventually(done).Should(Receive(BeNil()))
		},
		Entry("IPv4 TPROXY", SpecTimeout(10*time.Second), "tcp4", "198.51.100.10:18080", "0.0.0.0:17893", types.TargetTProxy),
		Entry("IPv6 TPROXY", SpecTimeout(10*time.Second), "tcp6", "[2001:db8::10]:18080", "[::]:17893", types.TargetTProxy),
		Entry("IPv6 DNS skips IPv4 DNAT", SpecTimeout(10*time.Second), "tcp6", "[2001:db8::53]:53", "[::]:17893", types.TargetTProxy),
		Entry("local service survives root DROP", SpecTimeout(10*time.Second), "tcp4", "172.31.254.1:18080", "0.0.0.0:18080", types.TargetDrop))

	It("delivers forwarded DNS to the DNAT server instead of the transparent proxy", func(ctx SpecContext) {
		tp := &config.TProxy{Name: "forwarded-dns", Port: 17893, Mark: 42042,
			DNSHijack: &config.DNSHijack{IP: pstr("172.31.254.1"), Port: 15353}}
		Expect(nft.AddChainAndRulesForTProxies([]*config.TProxy{tp})).To(Succeed())
		Expect(nft.SetRootTarget(types.Target{Op: types.TargetTProxy, Chain: "forwarded-dns-MARK"})).To(Succeed())
		listener, err := net.ListenPacket("udp4", "0.0.0.0:15353")
		Expect(err).To(Succeed())
		DeferCleanup(listener.Close)
		received := make(chan string, 2)
		go func() {
			buf := make([]byte, 128)
			for range 2 {
				n, addr, err := listener.ReadFrom(buf)
				if err != nil {
					return
				}
				received <- string(buf[:n])
				_, _ = listener.WriteTo(buf[:n], addr)
			}
		}()
		done := startForwardClient(ctx, "udp4", "198.51.100.53:53")
		Eventually(received).Should(Receive(Equal(forwardPayload)))
		Eventually(received).Should(Receive(Equal(forwardPayload)))
		Eventually(done).Should(Receive(BeNil()))
	}, SpecTimeout(10*time.Second))
})

const forwardPayload = "root cgroup forwarding\n"

// A separate process gives the client its own network namespace, like a bridge
// container, without changing the Go test runner's thread namespace.
func startForwardClient(ctx context.Context, protocol, address string) <-chan error {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestForwardClient$")
	cmd.Env = append(os.Environ(), "CGTPROXY_FORWARD_CLIENT="+protocol, "CGTPROXY_FORWARD_ADDRESS="+address)
	cmd.SysProcAttr = &syscall.SysProcAttr{Cloneflags: unix.CLONE_NEWNET}
	input, err := cmd.StdinPipe()
	Expect(err).To(Succeed())
	output, err := cmd.StdoutPipe()
	Expect(err).To(Succeed())
	var stderr strings.Builder
	cmd.Stderr = &stderr
	Expect(cmd.Start()).To(Succeed())
	DeferCleanup(func() { _ = cmd.Process.Kill() })
	ready, err := bufio.NewReader(output).ReadString('\n')
	Expect(err).To(Succeed())
	Expect(ready).To(Equal("ready\n"))

	veth := &netlink.Veth{LinkAttrs: netlink.LinkAttrs{Name: "cgp-host"},
		PeerName: "cgp-client", PeerNamespace: netlink.NsPid(cmd.Process.Pid)}
	Expect(netlink.LinkAdd(veth)).To(Succeed())
	DeferCleanup(func() { _ = netlink.LinkDel(veth) })
	addr, err := netlink.ParseAddr("172.31.254.1/24")
	Expect(err).To(Succeed())
	Expect(netlink.AddrAdd(veth, addr)).To(Succeed())
	Expect(netlink.LinkSetUp(veth)).To(Succeed())
	lo, err := netlink.LinkByName("lo")
	Expect(err).To(Succeed())
	Expect(netlink.LinkSetUp(lo)).To(Succeed())
	_, local, err := net.ParseCIDR("0.0.0.0/0")
	Expect(err).To(Succeed())
	defaultRoute := &netlink.Route{LinkIndex: veth.Index, Dst: local, Scope: unix.RT_SCOPE_LINK}
	Expect(netlink.RouteAdd(defaultRoute)).To(Succeed())
	DeferCleanup(func() { _ = netlink.RouteDel(defaultRoute) })
	route := &netlink.Route{LinkIndex: lo.Attrs().Index, Scope: unix.RT_SCOPE_HOST,
		Dst: local, Table: 30333, Type: unix.RTN_LOCAL}
	Expect(netlink.RouteAdd(route)).To(Succeed())
	DeferCleanup(func() { Expect(netlink.RouteDel(route)).To(Succeed()) })
	rule := netlink.NewRule()
	rule.Family, rule.Mark, rule.Table = netlink.FAMILY_V4, 42042, 30333
	Expect(netlink.RuleAdd(rule)).To(Succeed())
	DeferCleanup(func() { Expect(netlink.RuleDel(rule)).To(Succeed()) })
	if strings.HasSuffix(protocol, "6") {
		addr6, err := netlink.ParseAddr("fd42::1/64")
		Expect(err).To(Succeed())
		addr6.Flags = unix.IFA_F_NODAD
		Expect(netlink.AddrAdd(veth, addr6)).To(Succeed())
		_, local6, err := net.ParseCIDR("::/0")
		Expect(err).To(Succeed())
		default6 := &netlink.Route{LinkIndex: veth.Index, Dst: local6}
		Expect(netlink.RouteAdd(default6)).To(Succeed())
		DeferCleanup(func() { _ = netlink.RouteDel(default6) })
		route6 := &netlink.Route{LinkIndex: lo.Attrs().Index, Dst: local6, Table: 30333, Type: unix.RTN_LOCAL}
		Expect(netlink.RouteAdd(route6)).To(Succeed())
		DeferCleanup(func() { Expect(netlink.RouteDel(route6)).To(Succeed()) })
		rule6 := netlink.NewRule()
		rule6.Family, rule6.Mark, rule6.Table = netlink.FAMILY_V6, 42042, 30333
		Expect(netlink.RuleAdd(rule6)).To(Succeed())
		DeferCleanup(func() { Expect(netlink.RuleDel(rule6)).To(Succeed()) })
	}
	_, err = io.WriteString(input, "go\n")
	Expect(err).To(Succeed())
	Expect(input.Close()).To(Succeed())
	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			err = fmt.Errorf("%w: %s", err, stderr.String())
		}
		done <- err
	}()
	return done
}

func TestForwardClient(t *testing.T) {
	protocol := os.Getenv("CGTPROXY_FORWARD_CLIENT")
	if protocol == "" {
		return
	}
	fmt.Println("ready")
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	link, err := netlink.LinkByName("cgp-client")
	if err != nil {
		t.Fatal(err)
	}
	addr, err := netlink.ParseAddr("172.31.254.2/24")
	if err != nil {
		t.Fatal(err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		t.Fatal(err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		t.Fatal(err)
	}
	if err := netlink.RouteAdd(&netlink.Route{LinkIndex: link.Attrs().Index, Gw: net.ParseIP("172.31.254.1")}); err != nil {
		t.Fatal(err)
	}
	if strings.HasSuffix(protocol, "6") {
		addr6, err := netlink.ParseAddr("fd42::2/64")
		if err != nil {
			t.Fatal(err)
		}
		addr6.Flags = unix.IFA_F_NODAD
		if err := netlink.AddrAdd(link, addr6); err != nil {
			t.Fatal(err)
		}
		if err := netlink.RouteAdd(&netlink.Route{LinkIndex: link.Attrs().Index, Gw: net.ParseIP("fd42::1")}); err != nil {
			t.Fatal(err)
		}
	}
	conn, err := net.DialTimeout(protocol, os.Getenv("CGTPROXY_FORWARD_ADDRESS"), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	queries := 1
	if protocol == "udp4" {
		queries = 2 // Reuse conntrack's NAT binding, whose rules only ran once.
	}
	for range queries {
		if _, err := io.WriteString(conn, forwardPayload); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, len(forwardPayload))
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Fatal(err)
		}
		if string(buf) != forwardPayload {
			t.Fatalf("unexpected reply %q", buf)
		}
	}
}
