// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/ginkgo-helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

func TestRouteManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RouteManager Suite")
}

// fakeNFTManager is a test double for interfaces.NFTManager that records
// every call instead of touching the kernel.
type fakeNFTManager struct {
	addedRoutes  []types.Route
	removedPaths []string
	addedChains  []*config.TProxy

	inited   bool
	cleared  bool
	released bool

	initStructureErr error
	addChainErr      error
	addRoutesErr     error
	removeRoutesErr  error
	clearErr         error
	releaseErr       error
}

var _ interfaces.NFTManager = (*fakeNFTManager)(nil)

func (f *fakeNFTManager) InitStructure() error {
	f.inited = true
	return f.initStructureErr
}

func (f *fakeNFTManager) AddChainAndRulesForTProxies(tps []*config.TProxy) error {
	f.addedChains = append(f.addedChains, tps...)
	return f.addChainErr
}

func (f *fakeNFTManager) AddRoutes(routes []types.Route) error {
	f.addedRoutes = append(f.addedRoutes, routes...)
	return f.addRoutesErr
}

func (f *fakeNFTManager) RemoveRoutes(paths []string) error {
	f.removedPaths = append(f.removedPaths, paths...)
	return f.removeRoutesErr
}

func (f *fakeNFTManager) Clear() error {
	f.cleared = true
	return f.clearErr
}

func (f *fakeNFTManager) Release() error {
	f.released = true
	return f.releaseErr
}

// mustConfig builds a validated *config.Config from raw YAML, panicking the
// spec on failure.
func mustConfig(yamlContent string) *config.Config {
	cfg, err := config.New(config.WithContent([]byte(yamlContent)))
	Expect(err).ToNot(HaveOccurred())
	return cfg
}

const testConfigYAML = `
version: 1
cgroup-root: AUTO
route-table: 300
tproxies:
  clash:
    port: 7893
    mark: 520
rules:
  - match: .*proxy.*
    tproxy: clash
  - match: .*direct.*
    direct: true
  - match: .*drop.*
    drop: true
`

var _ = Describe("RouteManager", func() {
	Describe("construction via New", func() {
		Context("with a missing required dependency", func() {
			It("should fail when the NFT manager is nil", func() {
				_, err := New(WithConfig(mustConfig(testConfigYAML)), WithNFTMan(nil))
				Expect(err).To(MatchError(ErrNFTManagerMissing))
			})

			It("should fail when the config is nil", func() {
				_, err := New(WithConfig(nil))
				Expect(err).To(MatchError(ErrConfigMissing))
			})

			It("should fail when the event channel is nil", func() {
				_, err := New(WithCGroupEventChan(nil))
				Expect(err).To(MatchError(ErrCGroupEventChanMissing))
			})
		})

		Context("with all dependencies provided", func() {
			It("should compile the configured rules into matchers", func() {
				m, err := New(
					WithConfig(mustConfig(testConfigYAML)),
					WithNFTMan(&fakeNFTManager{}),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(m).ToNot(BeNil())
				Expect(m.matchers).To(HaveLen(3))
			})

			It("should accept a non-nil event channel", func() {
				ch := make(<-chan types.CGroupEvents)
				m, err := New(
					WithConfig(mustConfig(testConfigYAML)),
					WithNFTMan(&fakeNFTManager{}),
					WithCGroupEventChan(ch),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(m.cgroupEventsChan).To(BeIdenticalTo(ch))
			})

			It("should fall back to a nop logger when none is given", func() {
				m, err := New(
					WithConfig(mustConfig(testConfigYAML)),
					WithNFTMan(&fakeNFTManager{}),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(m.log).ToNot(BeNil())
			})

			It("should accept a custom logger", func() {
				log := zap.NewExample().Sugar()
				m, err := New(
					WithConfig(mustConfig(testConfigYAML)),
					WithNFTMan(&fakeNFTManager{}),
					WithLogger(log),
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(m.log).To(BeIdenticalTo(log))
			})
		})

		Context("with an invalid regex in a rule", func() {
			It("should fail to compile the matcher", func() {
				cfg := mustConfig(testConfigYAML)
				cfg.Rules[0].Match = "["

				_, err := New(WithConfig(cfg), WithNFTMan(&fakeNFTManager{}))
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("handleNewCgroups", func() {
		var (
			m   *RouteManager
			nft *fakeNFTManager
		)

		BeforeEach(func() {
			nft = &fakeNFTManager{}
			m, _ = New(
				WithConfig(mustConfig(testConfigYAML)),
				WithNFTMan(nft),
			)
		})

		ContextTable("matching cgroup path %q",
			ContextTableEntry("/user/proxy/app.service",
				types.TargetTProxy, "clash-MARK").WithFmt("/user/proxy/app.service"),
			ContextTableEntry("/user/direct/app.service",
				types.TargetDirect, "").WithFmt("/user/direct/app.service"),
			ContextTableEntry("/user/drop/app.service",
				types.TargetDrop, "").WithFmt("/user/drop/app.service"),
			func(path string, expectedOp types.TargetOp, expectedChain string) {
				It("should produce a single route with the expected target", func() {
					err := m.handleNewCgroups([]string{path})
					Expect(err).ToNot(HaveOccurred())

					Expect(nft.addedRoutes).To(HaveLen(1))
					route := nft.addedRoutes[0]
					Expect(route.Path).To(Equal(path))
					Expect(route.Target.Op).To(Equal(expectedOp))
					Expect(route.Target.Chain).To(Equal(expectedChain))
				})
			})

		Context("when no rule matches", func() {
			It("should not add any route", func() {
				err := m.handleNewCgroups([]string{"/nothing-matches-here"})
				Expect(err).ToNot(HaveOccurred())
				Expect(nft.addedRoutes).To(BeEmpty())
			})
		})

		Context("when several paths are handled in one batch", func() {
			It("should only add routes for the matching ones, in order", func() {
				paths := []string{
					"/user/proxy/a.service",
					"/user/ignored/b.service",
					"/user/drop/c.service",
				}
				err := m.handleNewCgroups(paths)
				Expect(err).ToNot(HaveOccurred())

				Expect(nft.addedRoutes).To(HaveLen(2))
				Expect(nft.addedRoutes[0].Path).To(Equal(paths[0]))
				Expect(nft.addedRoutes[0].Target.Op).To(Equal(types.TargetTProxy))
				Expect(nft.addedRoutes[1].Path).To(Equal(paths[2]))
				Expect(nft.addedRoutes[1].Target.Op).To(Equal(types.TargetDrop))
			})
		})

		Context("when the first matching rule wins", func() {
			It("should pick the tproxy rule over the direct rule", func() {
				err := m.handleNewCgroups([]string{"/user/proxy-and-direct.service"})
				Expect(err).ToNot(HaveOccurred())

				Expect(nft.addedRoutes).To(HaveLen(1))
				Expect(nft.addedRoutes[0].Target.Op).To(Equal(types.TargetTProxy))
			})
		})

		Context("when the NFT manager fails to add routes", func() {
			It("should propagate the wrapped error", func() {
				injected := errors.New("injected nft failure")
				nft.addRoutesErr = injected

				err := m.handleNewCgroups([]string{"/user/proxy/app.service"})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(injected))
			})
		})
	})

	Describe("handleDeleteCgroups", func() {
		var (
			m   *RouteManager
			nft *fakeNFTManager
		)

		BeforeEach(func() {
			nft = &fakeNFTManager{}
			m, _ = New(
				WithConfig(mustConfig(testConfigYAML)),
				WithNFTMan(nft),
			)
		})

		It("should forward every path to the NFT manager", func() {
			paths := []string{"/user/proxy/a.service", "/user/drop/b.service"}
			err := m.handleDeleteCgroups(paths)
			Expect(err).ToNot(HaveOccurred())
			Expect(nft.removedPaths).To(Equal(paths))
		})

		It("should propagate a removal failure", func() {
			injected := errors.New("injected remove failure")
			nft.removeRoutesErr = injected

			err := m.handleDeleteCgroups([]string{"/user/proxy/a.service"})
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(injected))
		})
	})
})

// RunRouteManager drives real netlink (ip rule / ip route) on top of the
// NFTManager interface, so it is exercised against the sandbox network
// namespace with a fake NFTManager injected. This covers addRoute, addRule,
// addRuleWithFamily, initializeNftableRuels, removeRoute, removeNftableRules
// and the event loop.
var _ = Describe("RunRouteManager (sandbox)", Ordered, func() {
	BeforeAll(func() {
		// Reuse the same gate as the nftman suite: it is set by `make test`
		// inside a fresh user+network namespace with CAP_NET_ADMIN.
		if !(os.Geteuid() == 0 && os.Getenv("CGTPROXY_TEST_NFTMAN") == "1") {
			Skip("RunRouteManager needs the sandbox network namespace; run via `make test`")
		}
	})

	var (
		m   *RouteManager
		nft *fakeNFTManager
		ch  chan types.CGroupEvents

		ctx        context.Context
		cancel     context.CancelFunc
		done       chan error
		tearedDown bool
	)

	BeforeEach(func() {
		// A fresh network namespace has its loopback interface down, which
		// makes "ip route add local default dev lo" fail with "network is
		// down". Bring it up to match a real system.
		lo, err := netlink.LinkByName("lo")
		Expect(err).ToNot(HaveOccurred())
		Expect(netlink.LinkSetUp(lo)).To(Succeed())

		nft = &fakeNFTManager{}
		ch = make(chan types.CGroupEvents)

		m, err = New(
			WithConfig(mustConfig(testConfigYAML)),
			WithNFTMan(nft),
			WithCGroupEventChan(ch),
		)
		Expect(err).ToNot(HaveOccurred())

		ctx, cancel = context.WithCancel(context.Background())
		done = make(chan error, 1)
		tearedDown = false
		go func() { done <- m.RunRouteManager(ctx) }()
	})

	AfterEach(func() {
		if !tearedDown {
			cancel()
			close(ch)
			Eventually(done, "5s").Should(Receive())
		}
	})

	It("should install routes/rules during startup and tear them down on exit", func() {
		By("initializing the nft structure and per-tproxy rules")
		// initializeNftableRuels calls InitStructure + AddChainAndRulesForTProxies
		// on the (fake) NFTManager before entering the event loop. Round-trip a
		// sentinel event through the loop to synchronize on setup completion.
		// The send runs in its own goroutine so a failed setup surfaces as a
		// clear timeout instead of a deadlock.
		result := make(chan error, 1)
		go func() {
			ch <- types.CGroupEvents{
				Events: []types.CGroupEvent{{
					Path:      "/user/proxy/app.service",
					EventType: types.CgroupEventTypeNew,
				}},
				Result: result,
			}
		}()
		Eventually(result, "5s").Should(Receive(BeNil()))

		Expect(nft.inited).To(BeTrue())
		Expect(nft.addedChains).To(HaveLen(1))
		Expect(nft.addedRoutes).To(HaveLen(1))
		Expect(nft.addedRoutes[0].Target.Op).To(Equal(types.TargetTProxy))

		By("removing the nft table and routes when the loop exits")
		// Closing the channel exits the for-range loop; cancelling the context
		// unblocks the trailing <-ctx.Done(). Both are required.
		close(ch)
		cancel()
		tearedDown = true

		var err error
		Eventually(done, "5s").Should(Receive(&err))
		Expect(err).To(MatchError(context.Canceled))

		Expect(nft.cleared).To(BeTrue())
		Expect(nft.released).To(BeTrue())
	})
})
