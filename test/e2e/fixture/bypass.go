// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/netip"
	"os"

	"github.com/google/nftables"
)

// The input lists disjoint, nonadjacent addresses. Check every start and end
// returned by the kernel: a truncated interval list can silently bypass traffic
// outside the configured addresses, rather than failing the transaction.
func checkBypassSet(args []string) error {
	flags := flag.NewFlagSet("check-bypass-set", flag.ExitOnError)
	name := flags.String("set", "bypass", "set name")
	addresses := flags.String("addresses", "", "file listing expected disjoint addresses")
	flags.Parse(args)
	file, err := os.Open(*addresses)
	if err != nil {
		return err
	}
	defer file.Close()
	want := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addr, err := netip.ParseAddr(scanner.Text())
		if err != nil {
			return err
		}
		want[string(addr.AsSlice())] = false
		want[string(addr.Next().AsSlice())] = true
		want[string(make([]byte, len(addr.AsSlice())))] = true
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	conn := &nftables.Conn{}
	set, err := conn.GetSetByName(&nftables.Table{Name: "cgtproxy", Family: nftables.TableFamilyINet}, *name)
	if err != nil {
		return err
	}
	elements, err := conn.GetSetElements(set)
	if err != nil {
		return err
	}
	if len(elements) != len(want) {
		return fmt.Errorf("%s has %d boundaries; want %d", *name, len(elements), len(want))
	}
	for _, element := range elements {
		end, ok := want[string(element.Key)]
		if !ok || element.IntervalEnd != end {
			return fmt.Errorf("%s: unexpected boundary %x, interval-end=%v", *name, element.Key, element.IntervalEnd)
		}
		delete(want, string(element.Key))
	}
	return nil
}
