// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/ginkgo-helper"
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
			func(err error) bool {
				var typeErr = &yaml.TypeError{}
				return errors.As(err, &typeErr)
			}, "yaml.TypeError",
		).WithFmt("../../../test/data/wrong_type.yaml"),
		ContextTableEntry(
			"../../../test/data/validation_fail.yaml",
			func(err error) bool {
				var validationErrs = validator.ValidationErrors{}
				return errors.As(err, &validationErrs)
			}, "validator.ValidationErrors",
		).WithFmt("../../../test/data/validation_fail.yaml"),
		func(path string, function func(error) bool, errString string) {
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
				Expect(err).To(MatchError(function, "Error should be a "+errString))
			})
		})
})

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}
