package config_test

import (
	"fmt"
	"io"
	"os"
	"testing"

	. "github.com/black-desk/deepin-network-proxy-manager/internal/config"
	. "github.com/black-desk/deepin-network-proxy-manager/pkg/ginkgo-helper"
	. "github.com/black-desk/deepin-network-proxy-manager/pkg/gomega-helper"
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Configuration", func() {
	ContextTable("load from valid configuration (%s)", func(path string) {
		var (
			err     error
			file    *os.File
			content []byte
		)

		BeforeEach(func() {
			file, err = os.Open(path)
			if err != nil {
				Fail(fmt.Sprintf("Failed to open configuration %s: %s", path, err.Error()))
			}

			content, err = io.ReadAll(file)
			if err != nil {
				Fail(fmt.Sprintf("Failed to read configuration %s: %s", path, err.Error()))
			}

			_, err = Load(content)
		})
		AfterEach(func() {
			file.Close()
		})
		It("should success.", func() {
			Expect(err).To(BeNil())
		})
	},
		ContextTableEntry("../../test/data/example_config.yaml"),
	)

	ContextTable("load from invalid configuration (%s)", func(path string, expectErr error) {
		var (
			err     error
			file    *os.File
			content []byte
		)

		BeforeEach(func() {
			file, err = os.Open(path)
			if err != nil {
				Fail(fmt.Sprintf("Failed to open configuration %s: %s", path, err.Error()))
			}

			content, err = io.ReadAll(file)
			if err != nil {
				Fail(fmt.Sprintf("Failed to read configuration %s: %s", path, err.Error()))
			}

			_, err = Load(content)
		})
		AfterEach(func() {
			file.Close()
		})

		It(fmt.Sprintf("should fail with error: %s", expectErr), func() {
			Expect(err).To(MatchErr(expectErr))
		})
	},
		ContextTableEntry("../../test/data/too_few_ports.yaml", ErrTooFewPorts).
			WithFmt("../../test/data/too_few_ports.yaml"),
		ContextTableEntry("../../test/data/wrong_type.yaml", new(yaml.TypeError)).
			WithFmt("../../test/data/wrong_type.yaml"),
		ContextTableEntry("../../test/data/validation_fail.yaml", validator.ValidationErrors{}).
			WithFmt("../../test/data/validation_fail.yaml"),
		ContextTableEntry("../../test/data/wrong_ports.yaml", new(ErrWrongPortsPattern)).
			WithFmt("../../test/data/wrong_ports.yaml"),
	)

})

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}
