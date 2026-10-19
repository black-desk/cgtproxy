#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

start_cgtproxy
for source in proxied forwarded; do
	probe "$source" tcp4 198.51.100.10:53 proxy
	probe "$source" udp4 198.51.100.10:53 origin
	probe "$source" tcp6 '[2001:db8::10]:53' proxy
	probe "$source" udp6 '[2001:db8::10]:53' origin
done
