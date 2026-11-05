// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package errwrap

import "fmt"

// Wrap adds context to a non-nil error, preserving its error chain.
// Use it with a named error result to annotate all failure exits of an operation:
//
//	defer errwrap.Wrap(&err, "load configuration")
//
// Nil errors and empty formats are left unchanged. Formatting arguments belong
// to the context only; the original error remains available to errors.Is/As.
func Wrap(err *error, format string, args ...any) {
	if err == nil || *err == nil || format == "" {
		return
	}
	*err = fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), *err)
}
