
GO ?= go
GOTAGS ?=

.PHONY: all
all:
	# https://github.com/google/wire/pull/353
	$(GO) get github.com/google/wire/cmd/wire@v0.5.0
	$(GO) generate -v -x ./...
	$(GO) mod tidy
	$(GO) build -v --tags=$(GOTAGS) ./cmd/cgtproxy

# We will create new cgroup dir in our tests,
# while current cgroup might not be owned by the user running test.
# That means by default, we should create new cgroup by systemd-run
# and run test in that cgroup.
SYSTEMD_RUN ?= systemd-run --user -d -P -t -q

# The tests code is written assuming that
# cgroup2 is mounted at /sys/fs/cgroup.
# So we have to unshare mount namespace to make sure that.
UNSHARE ?= unshare -U -C -m -n --map-user=0 --

SHELL ?= sh

CGROUPFS ?= /tmp/io.github.black-desk.cgtproxy-test/cgroupfs

COVERAGE ?= /tmp/io.github.black-desk.cgtproxy-test/coverage.out

.PHONY: test
test:
	$(SYSTEMD_RUN) \
	$(UNSHARE) \
	$(SHELL) -c "\
		mkdir -p $(CGROUPFS) && \
		mount --make-rprivate / && \
		mount -t cgroup2 none $(CGROUPFS) && \
		export CGTPROXY_TEST_CGROUP_ROOT=$(CGROUPFS) && \
		export CGTPROXY_TEST_NFTMAN=1 && \
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
