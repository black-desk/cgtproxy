.PHONY: \
	all \
	test \
	install

GO ?= go

all:
	$(GO) generate ./...
	$(GO) build ./cmd/cgtproxy

# We will create new cgroup dir in our tests,
# while current cgroup might not be owned by the user running test.
# That means by default, we should create new cgroup by systemd-run
# and run test in that cgroup.
SYSTEMD_RUN ?= systemd-run --user -d -P -q

# The tests code is written assuming that
# cgroup2 is mounted at /sys/fs/cgroup.
# So we have to unshare mount namespace to make sure that.
UNSHARE ?= unshare -U -C -m -n --map-user=0 --

test:
	$(SYSTEMD_RUN) \
	$(UNSHARE) \
	bash -c "\
		mount --make-rprivate / && \
		mount -t cgroup2 none /sys/fs/cgroup && \
		TEST_ALL=1 $(GO) test ./... -v --ginkgo.vv \
	"

PREFIX ?= /usr/local
DESTDIR ?= /

install:
	install -m755 -D cgtproxy \
		$(DESTDIR)$(PREFIX)/bin/cgtproxy
	install -m644 -D misc/systemd/cgtproxy.service \
		$(DESTDIR)$(PREFIX)/lib/systemd/system/cgtproxy.service
