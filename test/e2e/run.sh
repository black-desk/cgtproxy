#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

set -euo pipefail

suite=$(cd -- "$(dirname -- "$0")" && pwd)
export CGTPROXY_BIN=$1 FIXTURE_BIN=$2 GOCOVERDIR=$3
results=$(mktemp -d "$(dirname -- "$CGTPROXY_BIN")/results.XXXXXX")
delegated=/sys/fs/cgroup$(sed -n 's/^0:://p' /proc/self/cgroup)
failed=0
cases=0

for command in ip nft nsenter unshare timeout mount sysctl; do
	command -v "$command" >/dev/null || {
		echo "Missing test dependency: $command" >&2
		exit 1
	}
done

# A fresh cgroup and user/mount/network/PID namespace for every case and connector.
# Exiting the PID namespace also kills descendants if a case crashes or times out.
for reuse in true false; do
	for case_dir in "$suite"/cases/*; do
		name=$(basename -- "$case_dir")
		if [[ -n ${E2E_CASE:-} && $name != "$E2E_CASE" ]]; then
			continue
		fi
		export CASE_DIR=$case_dir CASE_WORK=$results/$name-$reuse REUSE_NETLINK=$reuse
		mkdir -p -- "$CASE_WORK"
		group=$delegated/e2e-$name-$reuse
		cases=$((cases + 1))
		mkdir -- "$group"
		if (
			echo "$BASHPID" >"$group/cgroup.procs"
			exec timeout --kill-after=5s 45s unshare -U -C -m -n -p -f --kill-child --map-user=0 --mount-proc \
				bash "$suite/sandbox.sh"
		) >"$CASE_WORK/case.log" 2>&1; then
			echo "PASS $name (reuse-netlink-socket=$reuse)"
		else
			echo "FAIL $name (reuse-netlink-socket=$reuse)" >&2
			cat -- "$CASE_WORK/case.log" >&2
			failed=1
		fi
		# Case-created cgroups remain after the namespace exits, but contain no tasks.
		find "$group" -depth -mindepth 1 -type d -exec rmdir -- '{}' \;
		rmdir -- "$group"
	done
done

echo "E2E logs: $results"
if ((cases == 0)); then
	echo "No case matched E2E_CASE=$E2E_CASE" >&2
	exit 1
fi
echo "$cases cases executed"
exit "$failed"
