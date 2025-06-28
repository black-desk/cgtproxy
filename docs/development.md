<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# Development notes

en | [zh_CN](./development.zh_CN.md)

<!-- Do not remove this warning when updating documentation -->

> [!WARNING]
>
> This English documentation is translated from the Chinese version using AI and
> may contain errors.

## Project Structure

This project uses [wire] to practice [dependency injection]. You need to check
the [wire.go] file to figure out how this application is constructed.

[wire]: https://github.com/google/wire
[dependency injection]: https://en.wikipedia.org/wiki/Dependency_injection
[wire.go]: ../cmd/cgtproxy/cmd/wire.go

Dependency injection is also used in tests, check [this](../pkg/nftman/wire.go).

All dependency of cgtproxy is exported as interface in [the interfaces package],
you are welcome to replace them with your own implementation.

[the interfaces package]: ../pkg/interfaces

The basic workflow of cgtproxy:

```text
fs notifier [github.com/rjeczalik/notify]
|
| new inotify event
v
cgroup monitor [github.com/black-desk/cgtproxy/pkg/cgfsmon]
|
| cgroup event
v
route manager [github.com/black-desk/cgtproxy/pkg/routeman]
|
| update nft, use nftman [github.com/black-desk/cgtproxy/pkg/nftman]
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

- When stderr is a terminal, log is written to stderr; otherwise, log is sent to
  journald.

- You can use environment variable `LOG_LEVEL` to control the log level,
  options: `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`.

- Some information is structured and printed in fields, you can use [fmtjournal]
  to view all log fields.

[fmtjournal]: https://github.com/black-desk/fmtjournal

## Build Tags

Add go build tag `debug` by `make GO_TAGS=debug` to produce debug build binary,
which:

1. Makes error carry more information like source position
2. Calls `nft` to print ruleset in logs after we update it
3. Adds debug counter to nft ruleset
