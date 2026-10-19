#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

set -euo pipefail

mount --make-rprivate /
mkdir "$CASE_WORK/cgroupfs"
mount -t cgroup2 none "$CASE_WORK/cgroupfs"
mount --bind "$CASE_WORK/cgroupfs" /sys/fs/cgroup
ip link set lo up

suite=$(cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=test/e2e/common.sh
source "$suite/common.sh"
trap cleanup EXIT

mkdir /sys/fs/cgroup/{servers,manager,direct,proxied,blocked,unmatched}
setup_network
start_servers
# Each case is a complete configuration and a script exercising the public CLI.
# shellcheck source=/dev/null
source "$CASE_DIR/test.sh"
stop_cgtproxy
