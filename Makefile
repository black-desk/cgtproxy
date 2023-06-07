.PHONY: \
	all \
	coverage \
	debug-build \
	cgtproxy \
	cgtproxy-debug \
	dlv-headless \
	generate \
	install \
	test-debug \
	test-release \
	test \
	test/coverage.html

all: cgtproxy

coverage: test/coverage.html

debug-build: cgtproxy-debug

cgtproxy: generate
	go build ./cmd/cgtproxy
	strip ./cgtproxy

cgtproxy-debug: generate
	go build -tags debug -o cgtproxy-debug \
		./cmd/cgtproxy

dlv-headless: generate
	dlv debug ./cmd/cgtproxy --headless

generate:
	go generate ./...

test-debug: generate
	unshare -U -C -m -n --map-user=0 -- bash -c "\
		mount --make-rprivate / && \
		mount -t cgroup2 none /sys/fs/cgroup && \
		go test ./... -tags debug -v --ginkgo.vv \
	"

test-release: generate
	unshare -U -C -m -n --map-user=0 -- bash -c "\
		mount --make-rprivate / && \
		mount -t cgroup2 none /sys/fs/cgroup && \
		go test ./... -v --ginkgo.vv \
	"

test: test-release

test/coverage.html: generate
	unshare -U -C -m -n --map-user=0 -- bash -c "\
		mount --make-rprivate / && \
		mount -t cgroup2 cgroup2 /sys/fs/cgroup && \
		( \
			env TEST_ALL=1 go test ./... -v --ginkgo.vv \
				-coverprofile test/coverprofile; \
			true \
		) \
	"
	go tool cover -html=test/coverprofile -o test/coverage.html

install: generate
	go install
