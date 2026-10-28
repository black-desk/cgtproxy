<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# Development notes

en | [zh_CN](./development.zh_CN.md)

<!-- Do not remove this warning when updating documentation -->

> [!WARNING]
>
> This English documentation is translated from the Chinese version using LLM
> and may contain errors.

## Building and Testing

See the [build guide](./build.md) for compilation, debug builds, test
environment requirements, commands, logs, and coverage.

## Project Structure

This project uses [wire] to practice [dependency injection]. You need to check
the [wire.go] file to figure out how this application is constructed.

[wire]: https://github.com/google/wire
[dependency injection]: https://en.wikipedia.org/wiki/Dependency_injection
[wire.go]: ../internal/cmd/wire.go

Implementation packages live in `internal/` and are private to cgtproxy.
Internal interfaces organize components and do not form a public library API.

The basic workflow of cgtproxy:

```text
fs notifier [github.com/fsnotify/fsnotify]
|
| new inotify event
v
cgroup monitor [github.com/black-desk/cgtproxy/internal/cgfsmon]
|
| cgroup event
v
route manager [github.com/black-desk/cgtproxy/internal/routeman]
|
| update nft, use nftman [github.com/black-desk/cgtproxy/internal/nftman]
v
netfilter framework in kernel
```

## Update NFTables Rule

Unlike the `nft` userspace util written in C, the golang implementation of
nftables by Google cannot execute nft scripts as `nft -f ...`, which makes us
have to figure out what low level expressions `nft` writes into netlink socket.

Refer to a [comment] from the author of that golang package, we could use
`nft --debug all -f ...` to check what is going on in `nft`.

[comment]: https://github.com/google/nftables/issues/5#issuecomment-451373151

I recommend using `nft --debug netlink -f ...` to check expressions written into
netlink socket, which helps you find out which structure in
`github.com/google/nftables/expr` you should use.

## Logging

`check config` enables logging by default and uses the same `--log-file` and
`LOG_LEVEL` settings as the main command.

- Use `--log-file /path/to/cgtproxy.log` to append JSON logs directly to a file
  instead of stderr or journald. New files are created with permissions `0600`;
  the parent directory must already exist.

- When stderr is a terminal, log is written to stderr; otherwise, log is sent to
  journald.

- You can use environment variable `LOG_LEVEL` to control the log level,
  options: `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`.

## Testing

See the [testing section of the build guide](./build.md#testing) for
instructions on running tests.

### How the Tests Work

Each [end-to-end case](../test/e2e/cases) runs in isolated user, mount, network,
PID, and cgroup namespaces and a delegated cgroup. The scripts create the
network topology, proxy servers, and nftables rules within this environment.

The test network contains a simulated host, a client attached through a bridge,
and a separate origin-server network namespace. The test proxy returns its
identity, the original destination, and the request payload so clients can
distinguish interception from direct access. Cases also verify dynamic cgroup
changes and removal of nftables tables, policy rules, and routes on graceful
shutdown.

Tests stop cgtproxy using SIGTERM so shutdown cleanup and coverage writes can
finish.

### Adding Cases

This project does not use unit tests.

To add an end-to-end test case, create a directory under `test/e2e/cases/`
containing a complete `config.yaml` and a `test.sh` script; the runner discovers
it automatically. Shared network setup and helpers such as `start_cgtproxy`,
`probe`, and `wait_until` are in [common.sh](../test/e2e/common.sh); the test
clients and transparent servers are in [fixture](../test/e2e/fixture/main.go).
Cases do not import internal packages or mock component interfaces.

When adding or fixing routing behavior, verify the expected result with real
traffic and add a regression scenario for the problem being fixed. Existing
cases cover configuration validation, local cgroup routing and dynamic updates,
gateway forwarding, DNS hijacking, protocol and address exclusions, and rule
cleanup on exit. Run tests for both regular and `debug` builds before
submitting. If a check cannot be run, document the reason and the validation
completed in the commit description or PR.
