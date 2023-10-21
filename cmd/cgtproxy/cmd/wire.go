//go:build wireinject
// +build wireinject

package cmd

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/google/wire"
	"go.uber.org/zap"
)

func injectedCGTProxy(
	*config.Config, *zap.SugaredLogger,
) (
	interfaces.CGTProxy, error,
) {
	panic(wire.Build(set))
}

func injectedLastingCGTProxy(
	*config.Config, *zap.SugaredLogger,
) (
	interfaces.CGTProxy, error,
) {
	panic(wire.Build(lastingConnectorSet))
}

var set = wire.NewSet(
	provideBypass,
	provideCGTProxy,
	provideCGroupEventChan,
	provideCgrougMontior,
	provideCgroupRoot,
	provideNFTManager,
	provideNetlinkConnector,
	provideRuleManager,
)

var lastingConnectorSet = wire.NewSet(
	provideBypass,
	provideCGTProxy,
	provideCGroupEventChan,
	provideCgrougMontior,
	provideCgroupRoot,
	provideLastringNetlinkConnector,
	provideNFTManager,
	provideRuleManager,
)
