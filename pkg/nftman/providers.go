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
	defer func() {
		if err != nil {
			return
		}

		err := connector.Delete()
		if err != nil {
			logger.Errorw(
				"Error delete netlink connector.",
				"error", err,
			)
		}

		return
	}()

	var man *NFTManager
	man, err = New(WithCgroupRoot(root), WithLogger(logger), WithConnFactory(connector))
	if err != nil {
		return
	}

	ret = man
	return
}
