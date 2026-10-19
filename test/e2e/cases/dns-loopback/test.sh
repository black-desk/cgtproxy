#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Omitting dns-hijack.ip exercises its 127.0.0.1 default.
sysctl -qw net.ipv4.conf.docker0.route_localnet=1
start_cgtproxy
probe forwarded udp4 198.51.100.10:53 dns
probe forwarded tcp4 198.51.100.10:53 dns
probe proxied udp4 198.51.100.10:53 dns
probe proxied tcp4 198.51.100.10:53 dns
