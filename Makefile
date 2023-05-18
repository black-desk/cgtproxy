.PHONY: \
	all \
	debug-build \
	dlv-headless \
	deepin-network-proxy-manager \
	deepin-network-proxy-manager-debug

all: deepin-network-proxy-manager

debug-build: deepin-network-proxy-manager-debug

dlv-headless:
	dlv debug ./cmd/deepin-network-proxy-manager --headless

deepin-network-proxy-manager:
	go build ./cmd/deepin-network-proxy-manager

deepin-network-proxy-manager-debug:
	go build -tags debug -o deepin-network-proxy-manager-debug \
		./cmd/deepin-network-proxy-manager
