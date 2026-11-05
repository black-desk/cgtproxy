// SPDX-FileCopyrightText: 2025-2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package routeman

import (
	"path/filepath"
	"strings"

	"github.com/black-desk/cgtproxy/internal/types"
)

// routesForPaths matches cgroup paths against configuration. It does not
// install kernel policy rules or routes; those operations are in routing.go.
func (m *RouteManager) routesForPaths(paths []string) []types.Route {
	routes := []types.Route{}
	root := filepath.Clean(string(m.cfg.CgroupRoot))

	for i := range paths {
		path := paths[i]

		m.log.Debugf("Checking routing configuration for cgroup %q.", path)

		// Match cgroup paths relative to cgroupfs, retaining the filesystem
		// path in the route for stat(2) and deletion handling.
		matchPath := strings.TrimPrefix(filepath.Clean(path), root)
		matchPath = "/" + strings.TrimPrefix(matchPath, "/")
		target := m.targetForPath(matchPath)

		if target.Op == types.TargetNoop {
			m.log.Debugf("No routing rule matches cgroup %q; inheriting the ancestor or root target.", matchPath)

			continue
		}

		routes = append(routes, types.Route{
			Path:   path,
			Target: target,
		})
	}

	return routes
}

func (m *RouteManager) targetForPath(path string) types.Target {
	for i := range m.matchers {
		if m.matchers[i].reg.MatchString(path) {
			m.log.Debugf("Cgroup %q matched rules[%d]: %s.", path, i, &m.cfg.Rules[i])
			return m.matchers[i].target
		}
	}
	return types.Target{}
}
