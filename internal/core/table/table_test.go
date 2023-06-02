package table_test

import (
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/table"
	tabletest "github.com/black-desk/deepin-network-proxy-manager/internal/core/table/internal/test"
	. "github.com/black-desk/lib/go/ginkgo-helper"
	"github.com/google/nftables"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func getNftRules() (res string) {
	out, err := exec.Command("nft", "list", "ruleset").Output()
	if err != nil {
		panic(err)
	}

	res = string(out)
	return
}

var _ = Describe("Netfliter table", Ordered, func() {
	var (
		err error
	)

	BeforeAll(func() {
		testAll := os.Getenv("TEST_ALL")
		if os.Geteuid() == 0 && testAll == "1" {
			return
		}

		Skip("" +
			"Skip tests of core/table as it requires some capabilities. " +
			"If you really want to run tests of this package, " +
			"try run `TEST_ALL=1 make test` or `TEST_ALL=1 make test-debug` " +
			"at the root directory of this repository.",
		)
	})

	Context("created", func() {
		var (
			conn *nftables.Conn
			t    *table.Table

			result string
		)

		BeforeEach(func() {
			By("Create a nftable connection.", func() {
				conn, err = tabletest.InjectedConn()
				Expect(err).To(Succeed())
			})

			By("Create a Table object.", func() {
				t, err = table.New(
					table.WithConn(conn),
					table.WithCgroupRoot(config.CgroupRoot("/sys/fs/cgroup")),
					table.WithRerouteMark(config.RerouteMark(1)),
				)
				Expect(err).To(Succeed())
			})
		})

		AfterEach(func() {
			By("Clear nftable content.", func() {
				if t == nil {
					return
				}
				err = t.Clear()
				Expect(err).To(Succeed())
			})

			By("Close nftable connection.", func() {
				err = conn.CloseLasting()
				Expect(err).To(Succeed())
			})
		})

		Context("then call Table.Clear()", func() {
			BeforeEach(func() {
				err = t.Clear()
			})

			It("should clear the nft table with no error", func() {
				Expect(err).To(Succeed())

				result = getNftRules()
				Expect(result).To(BeEmpty())
			})
		})

		type TproxyCase struct {
			t       *config.TProxy
			expects []string
		}

		ContextTable("with some tproxies", func(tps []*TproxyCase) {

			BeforeEach(func() {
				By("Initialize table with tproxies.", func() {
					for _, tp := range tps {
						err = t.AddChainAndRulesForTProxy(tp.t)
						Expect(err).To(Succeed(), "nft:\n%s", getNftRules())
					}
				})
			})

			It("should produce expected nftable rules", func() {
				result = getNftRules()

				Expect(result).To(ContainSubstring(consts.NftTableName))
				for _, tp := range tps {
					for _, expect := range tp.expects {
						Expect(result).To(ContainSubstring(expect))
					}
				}

			})

			Context("then add some cgroups", func() {
				BeforeEach(func() {
					err = os.MkdirAll("/sys/fs/cgroup/test/a", 0755)
					Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
					err = t.AddCgroup("/sys/fs/cgroup/test/a",
						&table.Target{Op: table.TargetTProxy, Chain: tps[rand.Intn(len(tps))].t.Name},
					)
					Expect(err).To(Succeed(), "nft:\n%s", getNftRules())

					err = os.MkdirAll("/sys/fs/cgroup/test/b", 0755)
					Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
					err = t.AddCgroup("/sys/fs/cgroup/test/b",
						&table.Target{Op: table.TargetTProxy, Chain: tps[rand.Intn(len(tps))].t.Name},
					)
					Expect(err).To(Succeed(), "nft:\n%s", getNftRules())

					err = os.MkdirAll("/sys/fs/cgroup/test/c", 0755)
					Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
					err = t.AddCgroup("/sys/fs/cgroup/test/c",
						&table.Target{Op: table.TargetDrop},
					)
					Expect(err).To(Succeed(), "nft:\n%s", getNftRules())

					err = os.MkdirAll("/sys/fs/cgroup/test/d/d", 0755)
					Expect(err).To(Or(Succeed(), MatchError(os.ErrExist)))
					err = t.AddCgroup("/sys/fs/cgroup/test/d/d",
						&table.Target{Op: table.TargetDirect},
					)
					Expect(err).To(Succeed(), "nft:\n%s", getNftRules())
				})

				AfterEach(func() {
					err = syscall.Rmdir("/sys/fs/cgroup/test/d/d")
					Expect(err).To(Succeed())
					err = syscall.Rmdir("/sys/fs/cgroup/test/d")
					Expect(err).To(Succeed())
					err = syscall.Rmdir("/sys/fs/cgroup/test/c")
					Expect(err).To(Succeed())
					err = syscall.Rmdir("/sys/fs/cgroup/test/b")
					Expect(err).To(Succeed())
					err = syscall.Rmdir("/sys/fs/cgroup/test/a")
					Expect(err).To(Succeed())
					err = syscall.Rmdir("/sys/fs/cgroup/test")
					Expect(err).To(Succeed())
				})

				It("should produce expected nftable rules", func() {
					result = getNftRules()
					{
						Expect(result).To(ContainSubstring(consts.NftTableName))
						for _, tp := range tps {
							Expect(result).To(ContainSubstring(tp.t.Name))
						}
						Expect(result).To(
							ContainSubstring("socket cgroupv2 level 3 @bypass-cgroup-3 return"),
						)
						Expect(result).To(
							ContainSubstring("socket cgroupv2 level 2 vmap @cgroup-map-2"),
						)
						Expect(result).To(
							ContainSubstring("test/d/d"),
						)
					}
				})

				Context("and remove them later", func() {
					BeforeEach(func() {
						t.RemoveCgroup("/sys/fs/cgroup/test/a")
						t.RemoveCgroup("/sys/fs/cgroup/test/b")
						t.RemoveCgroup("/sys/fs/cgroup/test/c")
						t.RemoveCgroup("/sys/fs/cgroup/test/d/d")
					})

					It("should produce expected nftable rules", func() {
						result = getNftRules()
						Expect(result).ToNot(ContainSubstring("jump"))
						Expect(result).ToNot(ContainSubstring("drop"))
					})

					Context("then add some of them back", func() {
						BeforeEach(func() {
							err = t.AddCgroup("/sys/fs/cgroup/test/a",
								&table.Target{Op: table.TargetTProxy, Chain: tps[rand.Intn(len(tps))].t.Name},
							)
							Expect(err).To(Succeed(), "nft:\n%s", getNftRules())
						})

						It("should produce expected nftable rules", func() {
							result = getNftRules()
							Expect(result).To(ContainSubstring("jump"))
							Expect(result).ToNot(ContainSubstring("drop"))
						})
					})
				})
			})
		},
			ContextTableEntry([]*TproxyCase{
				{
					t: &config.TProxy{
						Name:   "tproxy1",
						NoUDP:  true,
						NoIPv6: false,
						Port:   7893,
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
					},
					expects: []string{
						"chain tproxy4",
						"meta l4proto tcp tproxy ip to :7896",
					},
				},
			}).WithFmt(),
		)

	})

})

func TestTable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Table Suite")
}
