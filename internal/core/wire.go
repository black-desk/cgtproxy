//go:build wireinject
// +build wireinject

package core

import (
	"github.com/black-desk/cgtproxy/internal/core/monitor"
	"github.com/black-desk/cgtproxy/internal/core/rulemanager"
	"github.com/black-desk/cgtproxy/internal/core/watcher"
	"github.com/google/wire"
)

func injectedMonitor(*Core) (*monitor.Monitor, error) {
	panic(wire.Build(set))
}

func injectedRuleManager(*Core) (*rulemanager.RuleManager, error) {
	panic(wire.Build(set))
}

func injectedWatcher(*Core) (*watcher.Watcher, error) {
	panic(wire.Build(set))
}
