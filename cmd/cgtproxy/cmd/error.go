// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
)

type ErrCancelBySignal struct {
	os.Signal
}

func (e *ErrCancelBySignal) Error() string {
	return fmt.Sprintf("Cancelled by signal (%v).", e.Signal)
}
