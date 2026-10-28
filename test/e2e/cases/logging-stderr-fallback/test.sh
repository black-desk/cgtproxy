#!/bin/bash
# SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
#
# SPDX-License-Identifier: MIT

# Without journald, redirected stderr must still receive console logs.
unshare --mount bash -euc '
	mount --make-rprivate /
	mount -t tmpfs tmpfs /run
	LOG_LEVEL=info "$CGTPROXY_BIN" --config "$CASE_DIR/config.yaml" check config
' 2>"$CASE_WORK/no-journal.stderr"
grep -Eq $'\t(info|INFO)\tcgtproxy\t' "$CASE_WORK/no-journal.stderr"
if grep -q '^{' "$CASE_WORK/no-journal.stderr"; then
	echo 'Fallback stderr logs must use the console encoder' >&2
	exit 1
fi
