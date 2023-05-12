package location_test

import (
	"testing"

	. "github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Package `location`", func() {
	Context("capture a location in this file", func() {
    var location string
		BeforeEach(func() {
      location = Capture()
    })
		It("should contain file name \"location_test\".", func() {
      Expect(location).To(ContainSubstring("location_test"))
		})
	})
})

func TestLocation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Location Suite")
}
