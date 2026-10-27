//go:build debug

// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package errwrap

import (
	"fmt"
	"runtime"
)

// Wrap adds context and the caller's source position to a non-nil error,
// preserving its error chain.
func Wrap(err *error, format string, args ...any) {
	if err == nil || *err == nil {
		return
	}
	pc, file, line, _ := runtime.Caller(1)
	context := ""
	if format != "" {
		context = "\t" + fmt.Sprintf(format, args...) + "\n"
	}
	*err = fmt.Errorf("%s:%d (%x)\n%s%w", file, line, pc, context, *err)
}
