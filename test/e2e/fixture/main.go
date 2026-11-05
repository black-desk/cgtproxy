// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// fixture supplies clients, echo servers, and an embedded sing-box proxy.
// It deliberately does not import cgtproxy's implementation packages.
package main

import (
	"errors"
	"fmt"
	"os"
)

type receipt struct {
	Server      string `json:"server"`
	Destination string `json:"destination"`
	Payload     string `json:"payload"`
	Source      string `json:"source"`
}

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("usage: fixture serve|client|proxy [flags]"))
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2:])
	case "client":
		err = client(os.Args[2:])
	case "proxy":
		err = proxy(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
