// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package config

import (
	"fmt"
)

func (r *Rule) String() string {
	if r.Drop {
		return fmt.Sprintf("rule [ match: %s | DROP ]", r.Match)
	} else if r.Direct {
		return fmt.Sprintf("rule [ match: %s | DIRECT ]", r.Match)
	} else if r.TProxy != "" {
		return fmt.Sprintf("rule [ match: %s | TPROXY %s ]",
			r.Match, r.TProxy)
	}

	panic("this should never happened")
}
