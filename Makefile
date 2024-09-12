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

# Go variables
GO ?= $(shell goenv which go 2>/dev/null || command -v go)
GOTAGS ?=
_GO_TAGS =
GO_LDFLAGS ?=
_GO_LDFLAGS =
GO_MAIN_PACKAGE_DIR = ./cmd/cgtproxy

# Project version variables
# NOTE:
# These version variable initialization assumes that
# you are using POSIX compatible SHELL.
PROJECT_VERSION = 0.2.0
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
GO_VERSION_PACKAGE = github.com/black-desk/cgtproxy/cmd/cgtproxy/cmd
_GO_LDFLAGS += -X '$(GO_VERSION_PACKAGE).Version=v$(PROJECT_SEMVER_GENERATED_FROM_GIT_DESCRIBE)'
_GO_LDFLAGS += -X '$(GO_VERSION_PACKAGE).GitDescription=$(PROJECT_GIT_DESCRIBE)'

.PHONY: all
all:
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


# We will create new cgroup dir in our tests,
# while current cgroup might not be owned by the user running test.
# That means by default, we should create new cgroup by systemd-run
# and run test in that cgroup.
SYSTEMD_RUN ?= systemd-run --user -d -P -t -q

UNSHARE ?= unshare -U -C -m -n --map-user=0 --

CGROUPFS ?= /tmp/io.github.black-desk.cgtproxy-test/cgroupfs

GO_COVERAGE_OUTPUT ?= /tmp/io.github.black-desk.cgtproxy-test/coverage.out
GO_COVERAGE_REPORT ?= /tmp/io.github.black-desk.cgtproxy-test/coverage.txt

.PHONY: test
test:
	# NOTE:
	# Build tests but not run them.
	# Then you can run them without internet access.
	# The __SHOULD_NEVER_MATCH__ idea comes from
	# https://stackoverflow.com/a/72722257/13206417
	$(GO) test ./... \
		-ldflags "$(_GO_LDFLAGS) $(GO_LDFLAGS)" \
		-tags="$(_GO_TAGS),$(GO_TAGS)" \
		-run=__SHOULD_NEVER_MATCH__

	mkdir -p -- $(shell dirname -- "$(GO_COVERAGE_OUTPUT)")
	mkdir -p -- $(CGROUPFS)

	$(SYSTEMD_RUN) $(UNSHARE) $(SHELL) -c "\
		mount --make-rprivate / && \
		mount -t cgroup2 none $(CGROUPFS) && \
		export CGTPROXY_TEST_CGROUP_ROOT=$(CGROUPFS) && \
		export CGTPROXY_TEST_NFTMAN=1 && \
		export PATH=$(PATH) && \
		$(GO) test ./... -v --ginkgo.vv \
			-coverprofile=\"$(GO_COVERAGE_OUTPUT)\" \
			-ldflags \"$(_GO_LDFLAGS) $(GO_LDFLAGS)\" \
			-tags=\"$(_GO_TAGS),$(GO_TAGS)\" \
		"

.PHONY: test-coverage-report
test-coverage-report:
	go tool cover -func=$(GO_COVERAGE_OUTPUT) -o=$(GO_COVERAGE_REPORT)

.PHONY: install-bin
install-bin:
	$(INSTALL) -d "$(DESTDIR)$(bindir)"
	$(INSTALL_PROGRAM) cgtproxy "$(DESTDIR)$(bindir)"/cgtproxy

# List all paths that systemd will search for system level units using pkg-config
systemd_system_unit_paths=$(subst :, , $(shell \
	pkg-config systemd --variable=systemd_system_unit_path))
# Select the first path that is under $(libdir).
systemd_system_unit_dir?=$(word 1, \
	$(filter $(libdir)%, $(systemd_system_unit_paths)))

.PHONY: install-systemd-system-unit
install-systemd-system-unit:
ifeq ($(systemd_system_unit_dir),)
	$(error Failed to find location to install systemd system units)
endif
	$(INSTALL) -d "$(DESTDIR)$(systemd_system_unit_dir)"
	$(INSTALL_DATA) misc/systemd/cgtproxy.service \
		"$(DESTDIR)$(systemd_system_unit_dir)"/cgtproxy.service

.PHONY: install
install: install-bin install-systemd-system-unit
