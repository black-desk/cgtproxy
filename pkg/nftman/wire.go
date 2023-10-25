//go:build wireinject
// +build wireinject

package nftman

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/google/wire"
)

func injectedNFTManagerWithLastingConnector(config.CGroupRoot) (
	*NFTManager, error,
) {
	panic(wire.Build(testWithLastingConnectorSet))
}

func injectedNFTManagerWithConnector(config.CGroupRoot) (
	*NFTManager, error,
) {
	panic(wire.Build(testWithLastingConnectorSet))
}
