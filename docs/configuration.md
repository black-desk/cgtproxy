<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# Configuration Guide

en | [zh_CN](./configuration.zh_CN.md)

<!-- Do not remove this warning when updating documentation -->

> [!WARNING]
>
> This English documentation is translated from the Chinese version using AI and
> may contain errors.

You can refer to the [example configuration](../misc/config/example.yaml) and
[GoDoc][godoc].

[godoc]: https://pkg.go.dev/github.com/black-desk/cgtproxy

## Gateways

Besides `TPROXY` targets, cgtproxy can route the traffic of a cgroup to a
specific next-hop gateway, which is expected to do transparent proxy by
itself:

```yaml
gateways:
  home-router:
    ip: 192.168.1.1
    dev: eth0
    mark: 4000

rules:
  - match: /system.slice/foo.service
    gateway: home-router
```

Each gateway requires:

- `ip`: the address of the next-hop gateway (IPv4 or IPv6). Only the family of
  this address is routed to the gateway; traffic of the other family from a
  gateway-matched cgroup is sent directly.
- `dev`: the network interface used to reach the gateway. The gateway is
  treated as on-link, so no recursive route lookup is needed.
- `mark`: the fwmark used to select the route table of this gateway. It must
  be unique among all tproxies and gateways.

For each gateway, cgtproxy adds:

- an nft chain `<name>-MARK` which sets the fwmark;
- `ip rule add fwmark <mark> lookup <table>`;
- `ip route add default via <ip> dev <dev> table <table> onlink`.

Route tables are allocated automatically: the first gateway in name order gets
`route-table + 1`, the second gets `route-table + 2`, and so on.

> NOTE: The route table of a gateway only contains the default route via the
> gateway, so traffic to loopback or local addresses from a gateway-matched
> cgroup must be covered by `bypass`, otherwise it will be sent to the gateway
> as well.
