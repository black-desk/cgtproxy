# cgtproxy

cgtproxy is a transparent proxy rules manager written in go.
It will automatically update your nftable rules according to your configuration,
to archive per-app transparent proxy settings.

## How it works

Netfliter can be configured to filter network traffic [by cgroup],
as well as redirect some traffic to a [TPROXY] server.

[TPROXY]: https://www.infradead.org/~mchehab/kernel_docs/networking/tproxy.html
[by cgroup]: https://www.spinics.net/lists/netfilter/msg60360.html

Systemd has a work-in-progress XDG integration [documentation] suggest that
XDG applications should be launched in a systemd managed unit.

[documentation]: https://systemd.io/DESKTOP_ENVIRONMENTS

For example, telegram might be launched at some cgroup like
`/user.slice/user-1000.slice/user@1000.service/app.slice/app-flatpak-org.telegram.desktop@12345.service`

That means the cgroup path for the application has a pattern,
which we can match by a regex expression.

This program will listening cgroupfs change with inotify.
And update the nftable rules when new cgroup hierarchy created,
according to your configuration.

## Differences between cgproxy

This project is inspired by [cgproxy](https://github.com/springzfx/cgproxy).

But it has some differences:

- It use nftable;

  cgproxy using iptables, while cgtproxy use nftables.

- It use TPROXY only, no REDIR;

  cgproxy use TPROXY along with REDIR.

- It works more dynamically;

  cgproxy can only working with exsiting cgroup with fixed name.
  But cgtproxy can update nftables rules dynamically, for newly created cgroups.

- No BPF;

  cgproxy achieving per-app proxy setting by using BPF to trace syscall exec.
  cgtproxy implement this feature in the other way that doesn't required BPF.

- Less capabilities needed. (only CAP_NETWORK_ADMIN);

  To use BPF, more capabilities is required by cgproxy.
  But cgtproxy only require CAP_NETWORK_ADMIN
  to update route rules and nftables.

- Programs never get moved from original cgroup;

  cgproxy daemon will move processes
  with certain executable path into special cgroup.
  cgtproxy will never do that.

## Develop

Check this documentation [here](docs/development.md)
