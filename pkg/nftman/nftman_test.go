package nftman

import (
	"math/rand"
	"os"
	"syscall"
	"testing"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/ginkgo-helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Netfliter table", Ordered, func() {
	var (
		err error
	)

	BeforeAll(func() {
		testAll := os.Getenv("CGTPROXY_TEST_NFTMAN")
		if os.Geteuid() == 0 && testAll == "1" {
			return
		}

		Skip("" +
			"Skip tests of core/table as it requires some capabilities. " +
			"If you really want to run tests of this package, " +
			"try run `make test` at the root directory of this repository.",
		)
	})

	Context("created", func() {
		var (
			nft *NFTManager

			result     string
			cgroupRoot = os.Getenv("CGTPROXY_TEST_CGROUP_ROOT")
		)

		ContextTable("with %s",
			ContextTableEntry(injectedNFTManagerWithLastingConnector).WithFmt("lasting connector"),
			ContextTableEntry(injectedNFTManagerWithConnector).WithFmt("connector"),
			func(injectedNFTManager func(cGroupRoot config.CGroupRoot) (*NFTManager, error)) {
				BeforeEach(func() {
					By("Create a Table object and initialize structure", func() {
						nft, err = injectedNFTManager(config.CGroupRoot(cgroupRoot))
						Expect(err).To(Succeed())
						err = nft.InitStructure()
						Expect(err).To(Succeed())
					})
				})

				AfterEach(func() {
					By("Clear nftable content.", func() {
						if nft == nil {
							return
						}
						err = nft.Clear()
						Expect(err).To(Succeed())
					})
				})

				Context("then call Table.Clear()", func() {
					BeforeEach(func() {
						err = nft.Clear()
					})

					It("should clear the nft table with no error", func() {
						Expect(err).To(Succeed())

						result = getNFTableRules()
						Expect(result).To(BeEmpty())
					})
				})

				type TproxyCase struct {
					t       *config.TProxy
					expects []string
				}

				ContextTable("with some tproxies",
					ContextTableEntry([]*TproxyCase{
						{
							t: &config.TProxy{
								Name:   "tproxy1",
								NoUDP:  true,
								NoIPv6: false,
								Port:   7893,
								Mark:   100,
							},
							expects: []string{
								"chain tproxy1",
								"meta l4proto tcp tproxy to :7893",
							},
						},
						{
							t: &config.TProxy{
								Name:   "tproxy2",
								NoUDP:  false,
								NoIPv6: true,
								Port:   7894,
								Mark:   101,
							},
							expects: []string{
								"chain tproxy2",
								"meta l4proto { tcp, udp } tproxy ip to :7894",
							},
						},
						{
							t: &config.TProxy{
								Name:   "tproxy3",
								NoUDP:  false,
								NoIPv6: false,
								Port:   7895,
								Mark:   103,
							},
							expects: []string{
								"chain tproxy3",
								"meta l4proto { tcp, udp } tproxy to :7895",
							},
						},
						{
							t: &config.TProxy{
								Name:   "tproxy4",
								NoUDP:  true,
								NoIPv6: true,
								Port:   7896,
								Mark:   104,
							},
							expects: []string{
								"chain tproxy4",
								"meta l4proto tcp tproxy ip to :7896",
							},
						},
					}).WithFmt(),
					func(tps []*TproxyCase) {
						BeforeEach(func() {
							By("Initialize table with tproxies.", func() {
								tproxies := []*config.TProxy{}
								for _, tp := range tps {
									tproxies = append(tproxies, tp.t)
								}
								err = nft.AddChainAndRulesForTProxies(tproxies)
								Expect(err).To(Succeed(), "nft:\n%s", getNFTableRules())
							})
						})

						It("should produce expected nftable rules", func() {
							result = getNFTableRules()

							Expect(result).To(ContainSubstring(NftTableName))
							for _, tp := range tps {
								for _, expect := range tp.expects {
									Expect(result).To(ContainSubstring(expect))
								}
							}

						})

						Context("then add some cgroups", func() {
							BeforeEach(func() {
								err = os.MkdirAll(cgroupRoot+"/test/a", 0755)
								Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
								err = nft.AddRoutes([]types.Route{{Path: cgroupRoot + "/test/a",
									Target: types.Target{Op: types.TargetTProxy, Chain: tps[rand.Intn(len(tps))].t.Name + "-MARK"}}})
								Expect(err).To(Succeed(), "nft:\n%s", getNFTableRules())

								err = os.MkdirAll(cgroupRoot+"/test/b", 0755)
								Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
								err = nft.AddRoutes([]types.Route{{Path: cgroupRoot + "/test/b",
									Target: types.Target{Op: types.TargetTProxy, Chain: tps[rand.Intn(len(tps))].t.Name + "-MARK"}}})
								Expect(err).To(Succeed(), "nft:\n%s", getNFTableRules())

								err = os.MkdirAll(cgroupRoot+"/test/c", 0755)
								Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
								err = nft.AddRoutes([]types.Route{{Path: cgroupRoot + "/test/c",
									Target: types.Target{Op: types.TargetDrop}}})
								Expect(err).To(Succeed(), "nft:\n%s", getNFTableRules())

								err = os.MkdirAll(cgroupRoot+"/test/d/d", 0755)
								Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
								err = nft.AddRoutes([]types.Route{{Path: cgroupRoot + "/test/d/d",
									Target: types.Target{Op: types.TargetDirect}}})

								Expect(err).To(Succeed(), "nft:\n%s", getNFTableRules())
							})

							AfterEach(func() {
								err = syscall.Rmdir(cgroupRoot + "/test/d/d")
								Expect(err).To(Succeed())
								err = syscall.Rmdir(cgroupRoot + "/test/d")
								Expect(err).To(Succeed())
								err = syscall.Rmdir(cgroupRoot + "/test/c")
								Expect(err).To(Succeed())
								err = syscall.Rmdir(cgroupRoot + "/test/b")
								Expect(err).To(Succeed())
								err = syscall.Rmdir(cgroupRoot + "/test/a")
								Expect(err).To(Succeed())
								err = syscall.Rmdir(cgroupRoot + "/test")
								Expect(err).To(Succeed())
							})

							It("should produce expected nftable rules", func() {
								result = getNFTableRules()
								{
									Expect(result).To(ContainSubstring(NftTableName))
									for _, tp := range tps {
										Expect(result).To(ContainSubstring(tp.t.Name))
									}
									Expect(result).To(
										ContainSubstring("socket cgroupv2 level 3 vmap @cgroup-vmap"),
									)
									Expect(result).To(
										ContainSubstring("socket cgroupv2 level 2 vmap @cgroup-vmap"),
									)

									Expect(result).To(
										ContainSubstring(`test/d/d`),
									)

									Expect(result).To(
										ContainSubstring(`test/a" : goto tproxy`),
									)

									Expect(result).To(
										ContainSubstring(`test/b" : goto tproxy`),
									)

									Expect(result).To(
										ContainSubstring(`test/c" : drop`),
									)

									Expect(result).To(
										ContainSubstring(`test/d/d"`),
									)
								}
							})

							Context("and remove them later", func() {
								BeforeEach(func() {
									nft.RemoveRoutes([]string{
										cgroupRoot + "/test/a",
										cgroupRoot + "/test/b",
										cgroupRoot + "/test/c",
										cgroupRoot + "/test/d/d",
									})
								})

								It("should produce expected nftable rules", func() {
									result = getNFTableRules()
									Expect(result).ToNot(ContainSubstring("drop"))
								})

								Context("then add some of them back", func() {
									BeforeEach(func() {
										err = nft.AddRoutes([]types.Route{{Path: cgroupRoot + "/test/a",
											Target: types.Target{Op: types.TargetTProxy, Chain: tps[rand.Intn(len(tps))].t.Name + "-MARK"}}})
										Expect(err).To(Succeed(), "nft:\n%s", getNFTableRules())
									})

									It("should produce expected nftable rules", func() {
										result = getNFTableRules()
										Expect(result).To(ContainSubstring("goto"))
										Expect(result).ToNot(ContainSubstring("drop"))
									})
								})
							})
						})
					})
			})
	})
})

func TestTable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Table Suite")
}
