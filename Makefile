.PHONY: \
	all \
	coverage \
	debug-build \
	deepin-network-proxy-manager \
	deepin-network-proxy-manager-debug \
	dlv-headless \
	generate \
	install \
	test-debug \
	test-release \
	test \
	test/coverage.html

all: deepin-network-proxy-manager

coverage: test/coverage.html

debug-build: deepin-network-proxy-manager-debug

deepin-network-proxy-manager: generate
	go build ./cmd/deepin-network-proxy-manager
	strip ./deepin-network-proxy-manager

deepin-network-proxy-manager-debug: generate
	go build -tags debug -o deepin-network-proxy-manager-debug \
		./cmd/deepin-network-proxy-manager

dlv-headless: generate
	dlv debug ./cmd/deepin-network-proxy-manager --headless

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
