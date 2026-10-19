<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# Build Guide

en | [zh_CN](./build.zh_CN.md)

<!-- Do not remove this warning when updating documentation -->

> [!WARNING]
>
> This English documentation is translated from the Chinese version using AI and
> may contain errors.

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**,
**SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this
document are to be interpreted as described in [RFC 2119][rfc-2119].

[rfc-2119]: https://datatracker.ietf.org/doc/html/rfc2119

---

You **SHOULD** use `make` to build cgtproxy.

It is **NOT RECOMMENDED** to use `go build ./cmd/cgtproxy` to compile this
project.

## Testing

`make test` builds the complete cgtproxy executable with coverage and runs the
[end-to-end cases](../test/e2e/cases). Each case consists of `config.yaml` and
`test.sh` and runs in isolated user, mount, network, PID, and cgroup namespaces.
Every case runs with and without netlink socket reuse.

Tests require Linux, cgroup v2, a working systemd user session, Go, Bash,
iproute2, nftables, util-linux, and procps. `systemd-run --user` supplies a
writable delegated cgroup. The system must allow unprivileged user namespaces;
CI configures this permission.

```sh
make test
make test E2E_CASE=gateway-proxy
make test GO_TAGS=debug
```

The test network contains a simulated host, a client attached through a bridge,
and a separate origin-server network namespace. The test proxy returns its
identity, the original destination, and the request payload so clients can
distinguish interception from direct access. Drop checks require both a timeout
and no server receipt. Cases also verify dynamic cgroup changes and removal of
nft tables, policy rules, and routes on graceful shutdown.

Logs, server receipts, and network snapshots on failure are saved in
`.test/results.*`; raw coverage is saved in `.test/coverage.*`. `GOCOVERDIR`
collects coverage from the complete program, which is converted to the
Codecov-compatible `coverage.txt`. Set `GO_COVERAGE_PROFILE` to change the
output path. Tests stop cgtproxy using SIGTERM so shutdown cleanup and coverage
writes can finish.

To add a case, create a directory under `test/e2e/cases/` with a configuration
and script. Shared network setup and helpers are in
[common.sh](../test/e2e/common.sh); the test clients and transparent servers are
in [fixture](../test/e2e/fixture/main.go). Cases do not import internal packages
or mock component interfaces.
