<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# Build Guide

en | [zh_CN](./build.zh_CN.md)

<!-- Do not remove this warning when updating documentation -->

> [!WARNING]
>
> This English documentation is translated from the Chinese version using LLM
> and may contain errors.

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**,
**SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this
document are to be interpreted as described in [RFC 2119][rfc-2119].

[rfc-2119]: https://datatracker.ietf.org/doc/html/rfc2119

---

## Compilation

You **SHOULD** use `make` to build cgtproxy.

It is **NOT RECOMMENDED** to use `go build ./cmd/cgtproxy` to compile this
project.

## Debug Builds

Add go build tag `debug` by `make GO_TAGS=debug` to produce debug build binary,
which:

1. Makes error carry more information like source position
2. Calls `nft` to print ruleset in logs after we update it
3. Adds debug counter to nft ruleset

## Testing

Run the end-to-end tests from the repository root:

```bash
make test
make test GO_TAGS=debug
```

CI tests both regular and `debug` builds. `make test` builds cgtproxy with
coverage instrumentation, the test fixture, and a coverage tool. It then starts
cgtproxy through its public command-line interface and verifies routing with
real TCP/UDP traffic.

### Environment Requirements

- Use Linux with cgroup v2 and a working systemd user session. By default, tests
  run as a regular user through `systemd-run --user -d -P -q` to obtain a
  delegated cgroup.
- Install a Go toolchain satisfying `go.mod`, Make, Bash, and the commands
  provided by `iproute2`, `nftables`, `util-linux`, `coreutils`, and `procps`.
- The system must allow unprivileged user namespaces and support the cgroup,
  mount, network, and PID namespaces used by the tests. The kernel must also
  support nftables, TPROXY, IPv4/IPv6, bridges, and veth devices.

Traffic checks do not require an external proxy or internet connection; the
first build may need to download the Go toolchain and modules.

For errors such as `Operation not permitted`, unwritable cgroups, or failure to
connect to the user bus, check user namespace policies, the systemd user
session, and cgroup delegation first. On Ubuntu, CI also sets
`kernel.apparmor_restrict_unprivileged_userns=0` to allow the tests to create
user namespaces.

### Debugging and Results

Tests use `--log-file` to save application logs and default `LOG_LEVEL` to
`debug` unless it is already set.

Use `E2E_CASE` to select a single case directory under `test/e2e/cases/`:

```bash
make test E2E_CASE=local-routing
make test GO_TAGS=debug E2E_CASE=local-routing
```

Build artifacts default to `.test/`. Logs and server receipts for each run are
saved under `.test/results.*/<case-name>/`. Start with `case.log` and
`cgtproxy.log`. On failure, the scripts also attempt to save nftables rules,
policy routes, and neighbor tables for diagnosis.

Tests collect coverage from the complete program using `GOCOVERDIR`. Raw data is
saved in `.test/coverage.*` and converted to the Codecov-compatible
`coverage.txt`. Inspect it with the Go tools:

```bash
go tool cover -func=coverage.txt
go tool cover -html=coverage.txt
```

Override `TEST_BUILD_DIR` and `GO_COVERAGE_PROFILE` to change the artifact
directory and coverage report path.

See the [testing section of the development notes](./development.md#testing) for
how the tests work and requirements for adding cases.
