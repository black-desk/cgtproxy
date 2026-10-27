//go:build !debug

// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package errwrap

import "fmt"

// Wrap adds context to a non-nil error, preserving its error chain.
// An empty format leaves the error unchanged in regular builds.
func Wrap(err *error, format string, args ...any) {
	if err == nil || *err == nil || format == "" {
		return
	}
	*err = fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), *err)
}
