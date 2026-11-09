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

## Workflow

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

### CI Coverage Status

CI runs tests in multiple modes, including regular and `debug` builds, and
uploads their coverage reports separately. Codecov updates its status as these
reports arrive and may temporarily report a failure before all results are
available. Wait until all test jobs have finished and their coverage reports
have been processed before assessing the Codecov status.

### Adding Cases

This project does not use unit tests.

To add an end-to-end test case, create a directory under `test/e2e/cases/`
containing a complete `config.yaml` and a `test.sh` script; the runner discovers
it automatically. See [run.sh](../test/e2e/run.sh) for the runner and
[run-case.sh](../test/e2e/run-case.sh) for per-case initialization. Shared
network setup and helpers such as `start_cgtproxy`, `probe`, and `wait_until`
are in [common.sh](../test/e2e/common.sh); the test clients and ordinary echo
servers are in [fixture](../test/e2e/fixture/). Cases do not import internal
packages or mock components.

When adding or fixing routing behavior, verify the expected result with real
traffic and add a regression scenario for the problem being fixed. Existing
cases cover configuration validation, local cgroup routing and dynamic updates,
gateway forwarding, DNS hijacking, protocol and address exclusions, and rule
cleanup on exit. Run tests for both regular and `debug` builds before
submitting. If a check cannot be run, document the reason and the validation
completed in the commit description or PR.

## Releases

Both stable releases and prereleases are driven by `PROJECT_VERSION` in
`Makefile`. After a version change (for example, `0.3.4` or `0.3.4-rc.1`) is
committed to `master`, the `.github/workflows/cd.yaml` workflow automatically
handles the release process.
