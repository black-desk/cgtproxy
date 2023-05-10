package ginkgohelper_test

import (
	"testing"

	. "github.com/black-desk/deepin-network-proxy-manager/internal/test/ginkgo-helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ginkgo helper Suite")
}

var _ = Describe("ginkgo helper", func() {
	ContextTable("Given array %v",
		func(arg []int) {
			It("array[2]-array[1] should be a equal to 1", func() {
				Expect(arg[1] - arg[0]).To(Equal(1))
			})
		},
		ContextTableEntry([]int{1, 2}),
		ContextTableEntry([]int{2, 3}).WithFmt("2 3"),
	)
})
