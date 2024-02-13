// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package cmd

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/google/wire"
	"go.uber.org/zap"
)

// Injectors from wire.go:

func injectedCGTProxy(configConfig *config.Config, sugaredLogger *zap.SugaredLogger) (interfaces.CGTProxy, error) {
	cGroupRoot := provideCgroupRoot(configConfig)
	cGroupMonitor, err := provideCgrougMontior(cGroupRoot, sugaredLogger)
	if err != nil {
		return nil, err
	}
	netlinkConnector, err := provideNetlinkConnector()
	if err != nil {
		return nil, err
	}
	bypass := provideBypass(configConfig)
	nftManager, err := provideNFTManager(netlinkConnector, cGroupRoot, bypass, sugaredLogger)
	if err != nil {
		return nil, err
	}
	v := provideCGroupEventChan(cGroupMonitor)
	routeManager, err := provideRuleManager(nftManager, configConfig, v, sugaredLogger)
	if err != nil {
		return nil, err
	}
	cgtProxy, err := provideCGTProxy(cGroupMonitor, routeManager, sugaredLogger, configConfig)
	if err != nil {
		return nil, err
	}
	return cgtProxy, nil
}

func injectedLastingCGTProxy(configConfig *config.Config, sugaredLogger *zap.SugaredLogger) (interfaces.CGTProxy, error) {
	cGroupRoot := provideCgroupRoot(configConfig)
	cGroupMonitor, err := provideCgrougMontior(cGroupRoot, sugaredLogger)
	if err != nil {
		return nil, err
	}
	netlinkConnector, err := provideLastringNetlinkConnector()
	if err != nil {
		return nil, err
	}
	bypass := provideBypass(configConfig)
	nftManager, err := provideNFTManager(netlinkConnector, cGroupRoot, bypass, sugaredLogger)
	if err != nil {
		return nil, err
	}
	v := provideCGroupEventChan(cGroupMonitor)
	routeManager, err := provideRuleManager(nftManager, configConfig, v, sugaredLogger)
	if err != nil {
		return nil, err
	}
	cgtProxy, err := provideCGTProxy(cGroupMonitor, routeManager, sugaredLogger, configConfig)
	if err != nil {
		return nil, err
	}
	return cgtProxy, nil
}

// wire.go:

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
