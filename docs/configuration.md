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
[configuration fields](../internal/cgtproxy/config/config.go).

Each rule selects either `direct: true` or a named `tproxy`. Unknown
configuration fields, including the removed `drop` target, are rejected.
cgtproxy configures transparent proxies for desktop applications. It does not
provide network isolation or guarantee that traffic always uses a proxy.
Unexpected runtime errors cause it to report the failure, remove its rules and
routes, and exit. Cleanup failures are logged with the state that could not be
removed.

## Root cgroup

The path `/` is matched against `rules` in order. The first match selects the
root cgroup target, used as the fallback for local traffic. It supports `direct`
and `tproxy`. Explicit local cgroup decisions take precedence over the root
fallback.

## Gateway mode

Gateway mode handles traffic routed through this host, for example when it
serves as a router for LAN devices, forwards traffic for virtual machines, or
proxies outbound traffic from Docker bridge containers.

The top-level `gateway` setting defaults to `false`. When omitted or explicitly
set to `false`, forwarded traffic stays direct, even if the root target selects
a proxy. Local traffic still follows its existing rules. Set `gateway: true` to
apply the root target to traffic forwarded through this host from Docker bridge
containers, VMs, or LAN devices. For example:

```yaml
gateway: true
rules:
  - match: "^/$"
    tproxy: clash-meta
  # Other application rules...
```

A catch-all `match: '/.*'` also matches the root path and therefore controls
forwarded traffic when gateway mode is enabled. With no root match, forwarding
is left unchanged. An explicit `match: '^/$'` rule can also select a direct root
target. Forwarded packets cannot be matched against the sending process's
cgroup; all forwarding sources share the root target. `gateway` is a top-level
switch, not an option on an individual proxy in `tproxies`.

With gateway mode enabled, PREROUTING excludes connection replies, local
destinations, non-unicast traffic, and destinations in `bypass` before
processing TCP/UDP. The root proxy's `no-udp` and `no-ipv6` settings apply; a
configured IPv4 DNS hijack still handles UDP 53. DNS DNAT runs before TPROXY and
clears the proxy mark so the packet is routed to the DNS destination. Later
packets skip marking only when their original port is 53 and their DNAT
destination matches the selected proxy's DNS configuration. Other DNAT
connections continue to follow their cgroup or root target. If an external DNAT
rule changes the destination to an address in `bypass`, cgtproxy clears its
proxy mark and uses normal routing for every packet. Marks belonging to other
network components are preserved.

cgtproxy does not change forwarding sysctls, interface settings, or other
firewall rules. The network administrator supplies IP forwarding, firewall
permissions, and any NAT needed for direct forwarding; Docker usually configures
these.

If containers experience DNS timeouts after enabling gateway mode, see
[troubleshooting](./troubleshooting.md#docker-container-dns-times-out-in-gateway-mode).
