# Development notes

## project structure

    new underlying event (file system event from inoitfy for now)
    |
    | received by
    |
    cgroup monitor
    |
    | produce
    |
    cgroup event
    |
    | send to
    |
    rulemanager
    |
    | write rules to
    |
    nftable

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
