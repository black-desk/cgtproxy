SHELL=sh

GO ?= go
GOTAGS ?=
_GO_TAGS =
GO_LDFLAGS ?=
_GO_LDFLAGS =
GO_MAIN_PACKAGE_DIR = ./cmd/cgtproxy

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

GO_VERSION_PACKAGE = github.com/black-desk/cgtproxy/cmd/cgtproxy/cmd
_GO_LDFLAGS += -X '$(GO_VERSION_PACKAGE).Version=v$(PROJECT_SEMVER_GENERATED_FROM_GIT_DESCRIBE)'
_GO_LDFLAGS += -X '$(GO_VERSION_PACKAGE).GitDescription=$(PROJECT_GIT_DESCRIBE)'

.PHONY: all
all:
	$(GO) build -v \
		-ldflags "$(_GO_LDFLAGS) $(GO_LDFLAGS)" \
		-tags="$(_GO_TAGS),$(GO_TAGS)" \
		$(GO_MAIN_PACKAGE_DIR)

.PHONY: generate
generate:
	env GOWORK=off $(GO) generate -v -x ./...
	$(GO) mod tidy

# We will create new cgroup dir in our tests,
# while current cgroup might not be owned by the user running test.
# That means by default, we should create new cgroup by systemd-run
# and run test in that cgroup.
SYSTEMD_RUN ?= systemd-run --user -d -P -t -q

UNSHARE ?= unshare -U -C -m -n --map-user=0 --

CGROUPFS ?= /tmp/io.github.black-desk.cgtproxy-test/cgroupfs

COVERAGE ?= /tmp/io.github.black-desk.cgtproxy-test/coverage.out

.PHONY: test
test:
	# NOTE:
	# Build test binary before unshare.
	# As we unshare network namespace,
	# internet access will be lost after unshare is completed.
	# The __SHOULD_NEVER_MATCH__ idea comes from
	# https://stackoverflow.com/a/72722257/13206417
	$(GO) test ./... -tags=$(GOTAGS) -run=__SHOULD_NEVER_MATCH__

	$(SYSTEMD_RUN) \
	$(UNSHARE) \
	$(SHELL) -c "\
		mount --make-rprivate / && \
		mkdir -p $(CGROUPFS) && \
		mount -t cgroup2 none $(CGROUPFS) && \
		export CGTPROXY_TEST_CGROUP_ROOT=$(CGROUPFS) && \
		export CGTPROXY_TEST_NFTMAN=1 && \
		export PATH=$(PATH) && \
		mkdir -p $(shell dirname -- "$(COVERAGE)") && \
		$(GO) test ./... --tags=$(GOTAGS) -v --ginkgo.vv -coverprofile=$(COVERAGE) \
	"

PREFIX ?= /usr/local
DESTDIR ?=

.PHONY: install
install:
	install -m755 -D cgtproxy \
		$(DESTDIR)$(PREFIX)/bin/cgtproxy
	install -m644 -D misc/systemd/cgtproxy.service \
		$(DESTDIR)$(PREFIX)/lib/systemd/system/cgtproxy.service

COVERAGE_REPORT ?= /tmp/io.github.black-desk.cgtproxy-test/coverage.txt

.PHONY: test-coverage
test-coverage:
	go tool cover -func=$(COVERAGE) -o=$(COVERAGE_REPORT)
