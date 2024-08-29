
GO ?= go
GOTAGS ?=

.PHONY: all
all:
	env GOWORK=off $(GO) generate -v -x ./...
	$(GO) mod tidy
	$(GO) build -v --tags=$(GOTAGS) ./cmd/cgtproxy

# We will create new cgroup dir in our tests,
# while current cgroup might not be owned by the user running test.
# That means by default, we should create new cgroup by systemd-run
# and run test in that cgroup.
SYSTEMD_RUN ?= systemd-run --user -d -P -t -q

UNSHARE ?= unshare -U -C -m -n --map-user=0 --

SHELL ?= sh

CGROUPFS ?= /tmp/io.github.black-desk.cgtproxy-test/cgroupfs

COVERAGE ?= /tmp/io.github.black-desk.cgtproxy-test/coverage.out

.PHONY: test
test:
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
