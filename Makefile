# SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

DESTDIR?=
prefix?=/usr/local
bindir?=$(prefix)/bin
libdir?=$(prefix)/lib
libexecdir?=$(prefix)/libexec
datarootdir?=$(prefix)/share
INSTALL?=install
INSTALL_PROGRAM?= $(INSTALL)
INSTALL_DATA?= $(INSTALL) -m 644
SHELL=sh
PKG_CONFIG=pkg-config

# Go variables
GO ?= $(shell goenv which go 2>/dev/null || command -v go)
GO_TAGS ?=
_GO_TAGS =
GO_LDFLAGS ?=
_GO_LDFLAGS =
GO_MAIN_PACKAGE_DIR = ./cmd/cgtproxy

# Project version variables
# NOTE:
# These version variable initialization assumes that
# you are using POSIX compatible SHELL.
PROJECT_VERSION = 0.4.0-rc.1
PROJECT_GIT_DESCRIBE = $(shell git describe --tags --match v* --long --dirty)
PROJECT_SEMVER_GENERATED_FROM_GIT_DESCRIBE = $(shell \
	printf '%s\n' "$(PROJECT_GIT_DESCRIBE)" | \
	sed \
		-e 's/-\([[:digit:]]\+\)-g/+\1\./' \
		-e 's/-dirty$$/\.dirty/' \
		-e 's/+0\.[^\.]\+\.\?/+/' \
		-e 's/^v//' \
		-e 's/+$$//' \
)

# Integrate version string into golang -ldflags
GO_VERSION_PACKAGE = github.com/black-desk/cgtproxy/internal/cmd
_GO_LDFLAGS += -X '$(GO_VERSION_PACKAGE).Version=v$(PROJECT_SEMVER_GENERATED_FROM_GIT_DESCRIBE)'
_GO_LDFLAGS += -X '$(GO_VERSION_PACKAGE).GitDescription=$(PROJECT_GIT_DESCRIBE)'

.PHONY: all
all: generate
	$(GO) build -v \
		-ldflags "$(_GO_LDFLAGS) $(GO_LDFLAGS)" \
		-tags="$(_GO_TAGS),$(GO_TAGS)" \
		$(GO_MAIN_PACKAGE_DIR)

# `go generate` target
.PHONY: generate
generate:
	# NOTE:
	# Many developer write generate comamnds like
	# go run -mod=mod example.com/path/to/package
	# which is not working when go workspace is set and
	# update go.mod file.
	# So we need to disable workspace
	# and run go mod tidy after generate.
	env GOWORK=off $(GO) generate -v -x ./...
	$(GO) mod tidy


# Run each end-to-end case in a delegated cgroup and isolated namespaces.
SYSTEMD_RUN ?= systemd-run --user -d -P -q
TEST_BUILD_DIR ?= $(CURDIR)/.test
GO_COVERAGE_PROFILE ?= coverage.txt
E2E_CASE ?=

.PHONY: test
test:
	mkdir -p -- "$(TEST_BUILD_DIR)" "$$(dirname -- "$(GO_COVERAGE_PROFILE)")"
	$(GO) build -cover -coverpkg=./cmd/...,./internal/... -covermode=atomic \
		-ldflags "$(_GO_LDFLAGS) $(GO_LDFLAGS)" \
		-tags="$(_GO_TAGS),$(GO_TAGS)" \
		-o "$(TEST_BUILD_DIR)/cgtproxy" $(GO_MAIN_PACKAGE_DIR)
	$(GO) build -o "$(TEST_BUILD_DIR)/fixture" ./test/e2e/fixture
	# Building from GOROOT also supports toolchain downloads without covdata installed.
	$(GO) build -o "$(TEST_BUILD_DIR)/covdata" cmd/covdata
	set -e; \
		coverage_dir=$$(mktemp -d "$(TEST_BUILD_DIR)/coverage.XXXXXX"); \
		status=0; \
		$(SYSTEMD_RUN) env PATH="$(PATH)" E2E_CASE="$(E2E_CASE)" \
			bash test/e2e/run.sh "$(TEST_BUILD_DIR)/cgtproxy" "$(TEST_BUILD_DIR)/fixture" "$$coverage_dir" || status=$$?; \
		"$(TEST_BUILD_DIR)/covdata" textfmt -i="$$coverage_dir" -o="$(GO_COVERAGE_PROFILE)"; \
		exit "$$status"

.PHONY: install-bin
install-bin:
	$(INSTALL) -d "$(DESTDIR)$(bindir)"
	$(INSTALL_PROGRAM) cgtproxy "$(DESTDIR)$(bindir)"/cgtproxy

systemd_system_unit_dir?=$(shell \
	$(PKG_CONFIG) --define-variable=prefix=$(prefix) \
	systemd --variable=systemd_system_unit_dir)
ifeq ($(findstring $(systemd_system_unit_dir), $(subst :, , $(shell \
	$(PKG_CONFIG) systemd --variable=systemd_system_unit_path))), )
$(warning systemd_system_unit_dir="$(systemd_system_unit_dir)" \
	is not in the system unit search path of current systemd installation)
endif

.PHONY: install-systemd-system-unit
install-systemd-system-unit:
	$(INSTALL) -d "$(DESTDIR)$(systemd_system_unit_dir)"
	$(INSTALL_DATA) misc/systemd/cgtproxy.service \
		"$(DESTDIR)$(systemd_system_unit_dir)"/cgtproxy.service

.PHONY: install
install: install-bin install-systemd-system-unit
