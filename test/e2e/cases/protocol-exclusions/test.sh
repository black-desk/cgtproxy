#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
probe forwarded tcp4 198.51.100.10:18080 origin proxy
probe forwarded udp4 198.51.100.10:18080 origin direct
probe forwarded tcp6 '[2001:db8::10]:18080' origin direct
probe forwarded udp6 '[2001:db8::10]:18080' origin direct
probe forwarded udp4 198.51.100.10:53 dns direct 172.31.254.1:15353
probe forwarded tcp4 198.51.100.10:53 dns direct 172.31.254.1:15353
probe forwarded udp6 '[2001:db8::10]:53' origin direct

# Protocol exclusions also apply to explicit local cgroup routes.
probe proxied udp4 198.51.100.10:18080 origin direct
probe proxied udp6 '[2001:db8::10]:18080' origin direct
probe proxied udp4 198.51.100.10:53 dns direct 172.31.254.1:15353
probe proxied tcp6 '[2001:db8::10]:18080' origin direct
