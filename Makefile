.PHONY: \
	all \
	test \
	install

PREFIX ?= /usr/local
DESTDIR ?= /

all:
	go generate ./...
	go build ./cmd/cgtproxy

test:
	unshare -U -C -m -n --map-user=0 -- bash -c "\
		mount --make-rprivate / && \
		mount -t cgroup2 none /sys/fs/cgroup && \
		go test ./... -v --ginkgo.vv \
	"

install:
	install -m755 -D cgtproxy $(DESTDIR)$(PREFIX)/bin/cgtproxy
	install -m644 -D misc/systemd/cgtproxy.service $(DESTDIR)$(PREFIX)/lib/systemd/system/cgtproxy.service
