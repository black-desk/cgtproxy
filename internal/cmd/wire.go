// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build wireinject
// +build wireinject

package cmd

import (
	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/interfaces"
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

var set = wire.NewSet(
	provideBypass,
	provideCGTProxy,
	provideCGroupEventChan,
	provideCgrougMontior,
	provideCgroupRoot,
	provideGateway,
	provideNFTManager,
	provideNetlinkConnector,
	provideRuleManager,
)
