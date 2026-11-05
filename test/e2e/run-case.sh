#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

set -euo pipefail

# run.sh has already entered this case's namespaces. Mount cgroupfs again
# so its root reflects the cgroup namespace, then set up the shared network
# and services before executing the case's test.sh.
mkdir "$CASE_WORK/cgroupfs"
mount -t cgroup2 none "$CASE_WORK/cgroupfs"
mount --bind "$CASE_WORK/cgroupfs" /sys/fs/cgroup
ip link set lo up

suite=$(cd -- "$(dirname -- "$0")" && pwd)
# shellcheck source=test/e2e/common.sh
source "$suite/common.sh"
trap cleanup EXIT

# in_group places fixture services in servers, cgtproxy in manager, and
# local probes in the cgroups selected by each case's configuration.
mkdir /sys/fs/cgroup/{servers,manager,direct,proxied,unmatched}
setup_network
start_servers
# Each case is a complete configuration and a script exercising the public CLI.
# shellcheck source=/dev/null
source "$CASE_DIR/test.sh"
stop_cgtproxy
