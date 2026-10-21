// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"syscall"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/mdlayher/netlink"
)

// Observe transaction boundaries through the kernel's generation notifications.
// Reading the ruleset after startup alone would miss intermediate commits.
func checkNFTInit(args []string) error {
	flags := flag.NewFlagSet("check-nft-init", flag.ExitOnError)
	ready := flags.String("ready", "", "file created after subscribing to nftables events")
	cgroup := flags.String("cgroup", "", "cgroup whose entry must be in the initial transaction")
	flags.Parse(args)
	var key []byte
	if *cgroup != "" {
		info, err := os.Stat(*cgroup)
		if err != nil {
			return err
		}
		key = binaryutil.NativeEndian.PutUint64(info.Sys().(*syscall.Stat_t).Ino)
	}
	// A large initial transaction emits many multicast notifications at once.
	// Allow the observer to retain them until its monitor goroutine drains them.
	conn, err := nftables.New(nftables.WithSockOptions(func(conn *netlink.Conn) error {
		return conn.SetReadBuffer(4 << 20)
	}))
	if err != nil {
		return err
	}
	monitor := nftables.NewMonitor()
	transactions, err := conn.AddGenerationalMonitor(monitor)
	if err != nil {
		return err
	}
	defer monitor.Close()
	if err := os.WriteFile(*ready, nil, 0600); err != nil {
		return err
	}
	for transaction := range transactions {
		if transaction.GeneratedBy.Error != nil {
			return transaction.GeneratedBy.Error
		}
		created, fallback, root, direct := false, false, false, len(key) == 0
		for _, event := range transaction.Changes {
			if event.Error != nil {
				return event.Error
			}
			switch data := event.Data.(type) {
			case *nftables.Table:
				if event.Type == nftables.MonitorEventTypeNewTable && data.Name == "cgtproxy" {
					created = true
				}
			case *nftables.Rule:
				if event.Type != nftables.MonitorEventTypeNewRule || data.Table.Name != "cgtproxy" {
					continue
				}
				if data.Chain.Name == "cgroup-root" {
					root = true
				}
				for _, expression := range data.Exprs {
					if verdict, ok := expression.(*expr.Verdict); ok && data.Chain.Name == "output-mangle" &&
						verdict.Kind == expr.VerdictGoto && verdict.Chain == "cgroup-root" {
						fallback = true
					}
				}
			case []nftables.SetElement:
				for _, element := range data {
					if len(key) != 0 && bytes.Equal(element.Key, key) {
						direct = true
					}
				}
			}
		}
		if created {
			if !fallback || !root || !direct {
				return fmt.Errorf("incomplete initial transaction: output fallback=%v, root rule=%v, cgroup exclusion=%v", fallback, root, direct)
			}
			return nil
		}
	}
	return errors.New("nftables monitor closed before initialization")
}
