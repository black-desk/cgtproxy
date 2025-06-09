#!/bin/bash

# SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

set -e
set -x

# shellcheck disable=SC1090
. <(curl -fL "https://raw.githubusercontent.com/black-desk/get/master/get.sh") \
	black-desk cgtproxy

curl -fLO "https://raw.githubusercontent.com/black-desk/cgtproxy/v$VERSION/misc/systemd/cgtproxy.service"

$SUDO install -m755 -D "$TMP_DIR/cgtproxy" "$PREFIX/bin/cgtproxy"
$SUDO install -m644 -D "$TMP_DIR/cgtproxy.service" "$PREFIX/lib/systemd/system/cgtproxy.service"
