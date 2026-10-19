#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
probe forwarded tcp4 198.51.100.10:18080 proxy
probe forwarded udp4 198.51.100.10:18080 origin
probe forwarded tcp6 '[2001:db8::10]:18080' origin
probe forwarded udp6 '[2001:db8::10]:18080' origin
probe forwarded udp4 198.51.100.10:53 dns
probe forwarded tcp4 198.51.100.10:53 dns
probe forwarded udp6 '[2001:db8::10]:53' origin

# Protocol exclusions also apply to explicit local cgroup routes.
probe proxied udp4 198.51.100.10:18080 origin
probe proxied udp6 '[2001:db8::10]:18080' origin
probe proxied udp4 198.51.100.10:53 dns
probe proxied tcp6 '[2001:db8::10]:18080' origin
