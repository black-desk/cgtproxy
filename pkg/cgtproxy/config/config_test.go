package config_test

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/ginkgo-helper"
	. "github.com/black-desk/lib/go/gomega-helper"
	"github.com/go-playground/validator/v10"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

var _ = Describe("Configuration", func() {
	ContextTable("load from valid configuration (%s)",
		ContextTableEntry("../../../misc/config/example.yaml"),
		ContextTableEntry("../../../test/data/example_config.yaml"),
		func(path string) {
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

				_, err = config.New(config.WithContent(content))
			})
			AfterEach(func() {
				file.Close()
			})
			It("should success.", func() {
				Expect(err).To(BeNil())
			})
		})

	ContextTable("load from invalid configuration (%s)",
		ContextTableEntry(
			"../../../test/data/wrong_type.yaml",
			new(yaml.TypeError), "yaml.TypeError",
		).WithFmt("../../../test/data/wrong_type.yaml"),
		ContextTableEntry(
			"../../../test/data/validation_fail.yaml",
			validator.ValidationErrors{}, "validator.ValidationErrors",
		).WithFmt("../../../test/data/validation_fail.yaml"),
		func(path string, expectErr error, errString string) {
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

				_, err = config.New(config.WithContent(content))
			})

			AfterEach(func() {
				if file != nil {
					file.Close()
				}
			})

			It(fmt.Sprintf("should fail with error: %s", errString), func() {
				Expect(err).To(MatchErr(expectErr))
			})
		})
})

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}
