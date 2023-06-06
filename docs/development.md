# nftables rule development

Unlike the `nft` userspace util written in c, the golang implementation of
nftables by google is not aim to execute nft scripts as `nft -f ...`, which
makes we have to figure out what expression `nft` write into netlink socket.

Refer to a [comment][==link1==] from the author of that library, we could use
`nft --debug all -f ...` to check what is going on in `nft`.

I recommend use `nft --debug netlink -f ...` to check only the netlink level
expr written into netlink socket, which helps you find out which structure in
`github.com/google/nftables/expr` you should use.

[==link1==]: https://github.com/google/nftables/issues/5#issuecomment-451373151
