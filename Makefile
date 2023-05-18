.PHONY: deepin-network-proxy-manager deepin-network-proxy-manager-debug

all: deepin-network-proxy-manager

debug: deepin-network-proxy-manager-debug

deepin-network-proxy-manager:
	go build ./cmd/deepin-network-proxy-manager

deepin-network-proxy-manager-debug:
	go build -tags debug -o deepin-network-proxy-manager-debug ./cmd/deepin-network-proxy-manager
