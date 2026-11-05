// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter/certificate"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/dns/transport/local"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/direct"
	"github.com/sagernet/sing-box/protocol/redirect"
	"github.com/sagernet/sing/common/json"
)

func proxy(args []string) error {
	flags := flag.NewFlagSet("proxy", flag.ExitOnError)
	config := flags.String("config", "", "sing-box configuration file")
	ready := flags.String("ready", "", "file created after proxy startup")
	flags.Parse(args)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Register only the protocols used by the fixture. The other registries
	// are required by Box, but need no entries for this configuration.
	inbounds := inbound.NewRegistry()
	redirect.RegisterTProxy(inbounds)
	outbounds := outbound.NewRegistry()
	direct.RegisterOutbound(outbounds)
	transports := dns.NewTransportRegistry()
	local.RegisterTransport(transports)
	ctx = box.Context(ctx, inbounds, outbounds, endpoint.NewRegistry(),
		transports, service.NewRegistry(), certificate.NewRegistry())
	data, err := os.ReadFile(*config)
	if err != nil {
		return err
	}
	options, err := json.UnmarshalExtendedContext[option.Options](ctx, data)
	if err != nil {
		return fmt.Errorf("decode sing-box configuration: %w", err)
	}
	instance, err := box.New(box.Options{Context: ctx, Options: options})
	if err != nil {
		return fmt.Errorf("create sing-box: %w", err)
	}
	defer instance.Close()
	if err := instance.Start(); err != nil {
		return fmt.Errorf("start sing-box: %w", err)
	}
	if err := os.WriteFile(*ready, nil, 0600); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
