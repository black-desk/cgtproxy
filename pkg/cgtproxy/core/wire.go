//go:build wireinject
// +build wireinject

package core

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/google/wire"
	"go.uber.org/zap"
)

func injectedComponents(
	*config.Config, *zap.SugaredLogger,
) (
	*components, error,
) {
	panic(wire.Build(set))
}
