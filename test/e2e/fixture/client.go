// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"
)

func client(args []string) error {
	flags := flag.NewFlagSet("client", flag.ExitOnError)
	network := flags.String("network", "tcp4", "transport")
	address := flags.String("address", "198.51.100.10:18080", "destination")
	expect := flags.String("expect", "origin", "expected server identity")
	destination := flags.String("destination", "", "expected destination received by the server")
	source := flags.String("source-ip", "", "expected source IP reported by the server")
	count := flags.Int("count", 2, "requests on the same connection")
	flags.Parse(args)
	var sourceIP net.IP
	if *source != "" {
		sourceIP = net.ParseIP(*source)
		if sourceIP == nil {
			return fmt.Errorf("invalid expected source IP %q", *source)
		}
	}
	conn, err := net.DialTimeout(*network, *address, 3*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	decoder := json.NewDecoder(conn)
	for i := range *count {
		payload := fmt.Sprintf("cgtproxy-e2e-%d", i)
		if _, err := fmt.Fprintln(conn, payload); err != nil {
			return err
		}
		var value receipt
		if strings.HasPrefix(*network, "udp") {
			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil {
				return err
			}
			err = json.Unmarshal(buf[:n], &value)
		} else {
			err = decoder.Decode(&value)
		}
		if err != nil {
			return err
		}
		if sourceIP != nil {
			host, _, err := net.SplitHostPort(value.Source)
			if err != nil || !sourceIP.Equal(net.ParseIP(host)) {
				return fmt.Errorf("unexpected source %q; want IP %s", value.Source, *source)
			}
		}
		if value.Server != *expect || value.Payload != payload || (*destination != "" && value.Destination != *destination) {
			return fmt.Errorf("unexpected receipt %+v; want server=%s destination=%s payload=%s", value, *expect, *destination, payload)
		}
	}
	fmt.Printf("PASS %s %s -> %s\n", *network, *address, *expect)
	return nil
}
