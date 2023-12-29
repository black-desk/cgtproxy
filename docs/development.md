# Development notes

## project structure

This project uses [wire] to practice [dependency injection].

You need to check the [wire.go] file to figure out
how this application is constructed.

[wire]: https://github.com/google/wire
[dependency injection]: https://en.wikipedia.org/wiki/Dependency_injection
[wire.go]: ../cmd/cgtproxy/cmd/wire.go

Dependency injection is also used in tests, check [this](../pkg/nftman/wire.go).

All dependency of cgtproxy is exported as interface in [the interfaces package],
you are welcome to replace them with your own implementation.

[the interfaces package]: ../pkg/interfaces

    fs notifier [github.com/rjeczalik/notify]
    |
    | new inoitfy event
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

## nftables rule development

Unlike the `nft` userspace util written in c,
the golang implementation of nftables by google is not aim to
execute nft scripts as `nft -f ...`,
which makes we have to figure out
what low level expression `nft` write into netlink socket.

Refer to a [comment] from the author of that golang package,
we could use `nft --debug all -f ...` to check what is going on in `nft`.

I recommend use `nft --debug netlink -f ...` to
check expr written into netlink socket,
which helps you find out
which structure in `github.com/google/nftables/expr` you should use.

[comment]: https://github.com/google/nftables/issues/5#issuecomment-451373151

## log

When stderr is a terminal, log is write to stderr, else log is send to journald.

Environment variable `LOG_LEVEL` can use to control log level,
set it to `debug` or `trace` to enable more log.

Use [fmtjournal] to view all log fields.

[fmtjournal]: https://github.com/black-desk/fmtjournal

## build tags

Add go build tag `debug` by `make GOTAGS=debug` can
produce debug build binary, which:

1. make error carry more information like source position;
2. call to `nft` to dump ruleset after we update it;
3. add debug counter to nft ruleset.
