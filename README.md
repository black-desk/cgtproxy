# cgtproxy

`cgtproxy` is a transparent proxy **RULE** manager written in go
inspired by [cgproxy].

[cgproxy]: https://github.com/springzfx/cgproxy

It will automatically update your nft ruleset according to your configuration,
make it easier to archive per-app transparent proxy settings.

## The way how cgtproxy works.

Netfliter can be configured to filter network traffic [by cgroup],
as well as redirect some traffic to a [TPROXY] server.

[by cgroup]: https://www.spinics.net/lists/netfilter/msg60360.html
[TPROXY]: https://www.infradead.org/~mchehab/kernel_docs/networking/tproxy.html

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

## Why you might need such program?

On a linux desktop environment,
there are only few ways to configure network proxy settings at app level.

1. Set some network proxy environment variables,
   only for some applications.

   There is no elegant way to do this,
   but you can update the `.desktop` file of that application.

   But there is a problem that some applications
   might just ignore environment variables.

   That's why you might need a transparent network proxy setting.

2. If you using a proxy client such as clash,
   which can route packets based on the name of process
   that is sending the packet.

   Clash implement this feature by
   [going through procfs] when new connection created.

   [going through procfs]: https://github.com/Dreamacro/clash/blob/4d66da2277ddaf41f83bd889b064c0a584f7a8ad/component/process/process_linux.go#L129

   If there is a lot of processes,
   this implementation seems to have some performance issues.

   And if you need that executable,
   which you have configured to use proxy,
   temporarily connect to Internet directly.
   You have to update your clash configuration and restart clash,
   which means to close all old connections,
   which is quite annoying.

3. If your proxy client support [TPROXY], you can use [cgproxy].

   It can only update iptables for exsiting cgroup.

   For processes in cgroups that create later,
   it use BPF hooked on execve to match executable filename
   and move matched process to some other cgroup.

   This design has some serious problems:

   1. It will make processes removed from the original cgroup,
      even out of user slice.
   2. The `cgnoproxy` command it provided
      make any program can easily escape from original cgroup
      without any authentication.
   2. It create cgroup hierarchy without let systemd known.
      This behavior break the [single-writer rule]
      of design rules of the systemd cgroup API.

      [single-writer rule]: https://systemd.io/CGROUP_DELEGATION#two-key-design-rules

By using cgtproxy,
you can have flexible user-level per-app transparent network proxy settings
without any problems above.

## Differences between cgproxy

There are some differences between cgproxy and cgtproxy:

- cgproxy using iptables, but cgtproxy use nftables.

  <https://wiki.nftables.org/wiki-nftables/index.php/Main_differences_with_iptables>

- cgproxy can only working with exsiting cgroup,
  but cgtproxy can update nftables rules dynamically for new cgroups.

- cgproxy use BPF, but cgtproxy not;

  cgproxy implement per-app proxy by using BPF to trace the execve syscall.
  If the new executable file of that process matched some pattern,
  cgproxy daemon will put that process into a special hierarchy `proxy.slice`.

  This weird behavior make process escape from the user-level hierarchy,
  which means the systemd resource control settings will not take any effect.

  But cgtproxy implement per-app proxy by update nftable rules.
  It do not write anything to cgroupfs at all.

- cgproxy require at least CAP_NETWORK_ADMIN and CAP_BPF,
  but cgtproxy require only CAP_NETWORK_ADMIN.

  Check the [systemd service file] for details.

  [systemd service file]: ./misc/systemd/cgtproxy.service

## TODO

- [ ] optional cgroup monitor implementation listening on D-Bus
      instead of filesystem;
- [ ] DNS hijack for fake-ip;
  - [x] ipv4;
  - [ ] ipv6;
- [ ] ~~builtin TPROXY server.~~
