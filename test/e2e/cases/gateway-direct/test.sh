#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
for network in tcp4 udp4; do probe forwarded "$network" 198.51.100.10:18080 origin; done
for network in tcp6 udp6; do probe forwarded "$network" '[2001:db8::10]:18080' origin; done
probe proxied tcp4 198.51.100.10:18080 proxy
probe proxied udp6 '[2001:db8::10]:18080' proxy
