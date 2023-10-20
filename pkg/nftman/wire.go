//go:build wireinject
// +build wireinject

package nftman

import (
	"github.com/black-desk/cgtproxy/internal/tests/logger"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"

	"github.com/black-desk/cgtproxy/pkg/nftman/connector"
	"github.com/black-desk/cgtproxy/pkg/nftman/lastingconnector"
	"github.com/google/wire"
	"go.uber.org/zap"
)

func provideLastingConnector() (ret interfaces.NetlinkConnector, err error) {
	return lastingconnector.New()
}

func provideConnector() (ret interfaces.NetlinkConnector, err error) {
	return connector.New()
}

func provideNFTManager(
	root config.CGroupRoot, logger *zap.SugaredLogger, connector interfaces.NetlinkConnector,
) (
	ret *NFTManager, err error,
) {

	return New(WithCgroupRoot(root), WithLogger(logger), WithConnFactory(connector))

}

var testWithLastingConnectorSet = wire.NewSet(
	provideLastingConnector,
	provideNFTManager,
	logger.ProvideLogger,
)

var testWithConnectorSet = wire.NewSet(
	provideConnector,
	provideNFTManager,
	logger.ProvideLogger,
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
