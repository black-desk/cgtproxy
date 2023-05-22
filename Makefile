.PHONY: \
	all \
	coverage \
	debug-build \
	deepin-network-proxy-manager \
	deepin-network-proxy-manager-debug \
	dlv-headless \
	generate \
	install \
	test

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

test: generate
	go test ./...

test/coverage.html: generate
	go test ./... -coverprofile test/coverprofile
	go tool cover -html=test/coverprofile -o test/coverage.html

install: generate
	go install
