//go:build wireinject
// +build wireinject

package core

import (
	"github.com/black-desk/cgtproxy/internal/config"
	"github.com/google/wire"
)

func injectedComponents(*config.Config) (*components, error) {
	panic(wire.Build(set))
}
