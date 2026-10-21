<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# Troubleshooting

en | [zh_CN](./troubleshooting.zh_CN.md)

<!-- Do not remove this warning when updating documentation -->

> [!WARNING]
>
> This English documentation is translated from the Chinese version using LLM
> and may contain errors.

## File exists

```text
Error:
running cgtproxy core: running route manager: add route: file exists
running filesystem watcher: context canceled
```

The "file exists" is error message of EEXIST, which means that:

1. Maybe there is another cgtproxy running;

   Try to stop another cgtproxy.

2. Previous cgtproxy doesn't exit normally, perhaps killed with SIG_KILL.

   A system reboot should fix this issue, if you don't want to do so read the
   instructions below:
   1. Check what is exactly going on:

      ```bash
      # List route rules in the table cgtproxy created.
      # 300 is the default value of "route-table" in configuration.
      ip route list table 300
      # > local default dev lo scope host

      # List nft ruleset created by cgtproxy.
      sudo nft list table inet cgtproxy
      # > table inet cgtproxy {
      # >   set bypass {
      # >           type ipv4_addr
      # >           flags interval
      # > ...

      # Make sure cgtproxy is not running.
      sudo lsof /usr/local/bin/cgtproxy 2>/dev/null
      # No output
      ```

   2. Now you can just simply remove the route table and nftable ruleset by:

      ```bash
      sudo ip route del table 300 local default dev lo scope host
      sudo ip -6 route del table 300 local default dev lo scope host
      sudo ip rule del fwmark 3000 lookup 300
      sudo ip -6 rule del fwmark 3000 lookup 300
      sudo nft flush table inet cgtproxy
      sudo nft delete table inet cgtproxy
      ```

## Docker container DNS times out in gateway mode

If
[gateway mode](https://pkg.go.dev/github.com/black-desk/cgtproxy/internal/cgtproxy/config#Config.Gateway)
is enabled and the selected proxy's `dns-hijack.ip` is an IPv4 loopback address
such as `127.0.0.1`, container DNS requests are DNATed to the host's loopback
address. The kernel normally rejects routing packets arriving from other
interfaces to loopback addresses, so DNS requests may be dropped even after they
match the hijack rules.

In this situation, enable `route_localnet` on the host's incoming interface. See
the comments before `dns-hijack` in the
[example configuration](../misc/config/example.yaml) for commands, persistence,
and interface creation timing. For custom bridges, replace `docker0` with the
actual incoming interface.

## DNS Resolution Not Being Redirected

If you find that DNS requests from certain programs (such as `curl`) are not
being redirected, this may be caused by the NSS (Name Service Switch) mechanism.

When programs use the NSS (Name Service Switch) functionality provided by libc
for domain name resolution, they check the `hosts` line in the
`/etc/nsswitch.conf` configuration file. In some distributions, this line
contains a `resolve` entry before `dns`:

```text
hosts: mymachines resolve [!UNAVAIL=return] files myhostname dns
```

This `resolve` entry corresponds to `libnss-resolve.so` (an NSS plugin provided
by systemd), which **connects directly to `systemd-resolved` via local socket**
instead of sending DNS queries through the network stack. Therefore, these DNS
requests completely bypass nftables rules and cannot be redirected by cgtproxy.

> [!WARNING] The following solution requires modifying system configuration. Do
> NOT make this modification unless you understand what it means and the
> potential consequences.

Consider removing `resolve` from the `hosts` line in `/etc/nsswitch.conf`. This
will make programs use standard DNS queries sent through the network stack,
allowing them to be captured by netfilter rules.

In most cases currently, this has the same effect as using the `resolve` plugin:
when systemd-resolved is enabled, `/etc/resolv.conf` is taken over by
systemd-resolved (usually pointing to `127.0.0.53`), so programs will still
perform DNS resolution through systemd-resolved, but this time through the
network stack.
