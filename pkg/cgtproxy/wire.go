//go:build wireinject
// +build wireinject

package cgtproxy

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/google/wire"
	"go.uber.org/zap"
)

var set = wire.NewSet(
	provideBypass,
	provideCgrougMontior,
	provideCgroupRoot,
	provideChans,
	provideComponents,
	provideInputChan,
	provideOutputChan,
	provideRuleManager,
	provideNFTMan,
)

func injectedComponents(
	*config.Config, *zap.SugaredLogger,
) (
	*components, error,
) {
	panic(wire.Build(set))
}
