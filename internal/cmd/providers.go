// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/black-desk/cgtproxy/internal/cgfsmon"
	"github.com/black-desk/cgtproxy/internal/cgtproxy"
	"github.com/black-desk/cgtproxy/internal/cgtproxy/config"
	"github.com/black-desk/cgtproxy/internal/interfaces"
	"github.com/black-desk/cgtproxy/internal/nftman"
	"github.com/black-desk/cgtproxy/internal/nftman/connector"
	"github.com/black-desk/cgtproxy/internal/routeman"
	"github.com/black-desk/cgtproxy/internal/types"
	"go.uber.org/zap"
)

func provideCGroupEventChan(mon interfaces.CGroupMonitor) <-chan types.CGroupEvents {
	return mon.Events()
}

func provideNetlinkConnector() (ret interfaces.NetlinkConnector, err error) {
	return connector.New()
}

func provideNFTManager(
	connector interfaces.NetlinkConnector,
	root config.CGroupRoot,
	bypass config.Bypass,
	gateway bool,
	logger *zap.SugaredLogger,
) (
	ret interfaces.NFTManager,
	err error,
) {
	return nftman.New(
		nftman.WithCgroupRoot(root),
		nftman.WithBypass(bypass),
		nftman.WithGateway(gateway),
		nftman.WithLogger(logger),
		nftman.WithConnFactory(connector),
	)
}

func provideRuleManager(
	t interfaces.NFTManager,
	cfg *config.Config,
	ch <-chan types.CGroupEvents,
	logger *zap.SugaredLogger,
) (
	ret interfaces.RouteManager, err error,
) {
	return routeman.New(
		routeman.WithNFTMan(t),
		routeman.WithConfig(cfg),
		routeman.WithCGroupEventChan(ch),
		routeman.WithLogger(logger),
	)
}

func provideCgrougMontior(
	cgroupRoot config.CGroupRoot,
	logger *zap.SugaredLogger,
) (
	interfaces.CGroupMonitor, error,
) {
	return cgfsmon.New(
		cgfsmon.WithCgroupRoot(cgroupRoot),
		cgfsmon.WithLogger(logger),
	)
}

func provideCgroupRoot(cfg *config.Config) config.CGroupRoot {
	return cfg.CgroupRoot
}

func provideBypass(cfg *config.Config) config.Bypass {
	return cfg.Bypass
}

func provideGateway(cfg *config.Config) bool {
	return cfg.Gateway
}

func provideCGTProxy(
	mon interfaces.CGroupMonitor,
	man interfaces.RouteManager,
	logger *zap.SugaredLogger,
	cfg *config.Config,
) (
	interfaces.CGTProxy, error,
) {
	return cgtproxy.New(
		cgtproxy.WithConfig(cfg),
		cgtproxy.WithLogger(logger),
		cgtproxy.WithCGroupMonitor(mon),
		cgtproxy.WithRouteManager(man),
	)
}
