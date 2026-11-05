// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func serve(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	network := flags.String("network", "tcp4", "tcp4, tcp6, udp4 or udp6")
	address := flags.String("address", "127.0.0.1:18080", "listen address")
	id := flags.String("id", "origin", "server identity returned to clients")
	ready := flags.String("ready", "", "file created after listening")
	flags.Parse(args)
	var mu sync.Mutex
	record := func(value receipt) {
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewEncoder(os.Stdout).Encode(value)
	}
	markReady := func() error { return os.WriteFile(*ready, nil, 0600) }
	if strings.HasPrefix(*network, "tcp") {
		listener, err := net.Listen(*network, *address)
		if err != nil {
			return err
		}
		defer listener.Close()
		if err := markReady(); err != nil {
			return err
		}
		for {
			conn, err := listener.Accept()
			if err != nil {
				return err
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(
					time.Now().Add(5 * time.Second),
				)
				scanner := bufio.NewScanner(conn)
				for scanner.Scan() {
					value := receipt{
						*id,
						conn.LocalAddr().String(),
						scanner.Text(),
						conn.RemoteAddr().String(),
					}
					record(value)
					err := json.NewEncoder(conn).Encode(value)
					if err != nil {
						return
					}
				}
				if err := scanner.Err(); err != nil {
					fmt.Fprintf(
						os.Stderr,
						"Read TCP request from %s: %v\n",
						conn.RemoteAddr(),
						err,
					)
				}
			}()
		}
	}
	conn, err := net.ListenPacket(*network, *address)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Bind each UDP echo server to a concrete destination. Its local address
	// then identifies the target and WriteTo replies from the correct IP/port.
	if conn.LocalAddr().(*net.UDPAddr).IP.IsUnspecified() {
		return errors.New("UDP echo server requires a concrete listen address")
	}
	if err := markReady(); err != nil {
		return err
	}
	buf := make([]byte, 4096)
	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}
		value := receipt{
			Server:      *id,
			Destination: conn.LocalAddr().String(),
			Payload:     strings.TrimSuffix(string(buf[:n]), "\n"),
			Source:      peer.String(),
		}
		record(value)
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if _, err := conn.WriteTo(append(data, '\n'), peer); err != nil {
			return err
		}
	}
}
