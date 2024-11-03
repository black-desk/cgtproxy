# `cgtproxy`

- ![license][shields-license]
- ![checks][shields-checks]
  [![codecov][shields-codecov]][codecov]
  [![FOSSA License Scan Status][shields-fossa-license]][fossa-license]
  [![FOSSA Security Scan Status][shields-fossa-security]][fossa-security]
- ![commit activity][shields-commit-activity]
  ![contributors][shields-contributors]
  ![release date][shields-release-date]
  ![commits since release][shields-commits-since-release]

[shields-license]: https://img.shields.io/github/license/black-desk/cgtproxy
[shields-checks]: https://img.shields.io/github/check-runs/black-desk/cgtproxy/master
[shields-codecov]: https://codecov.io/github/black-desk/cgtproxy/graph/badge.svg?token=6TSVGQ4L9X
[codecov]: https://codecov.io/github/black-desk/cgtproxy
[shields-fossa-license]: https://app.fossa.com/api/projects/custom%2B48828%2Fgithub.com%2Fblack-desk%2Fcgtproxy.svg?type=shield&issueType=license
[fossa-license]: https://app.fossa.com/projects/custom%2B48828%2Fgithub.com%2Fblack-desk%2Fcgtproxy?ref=badge_shield&issueType=license
[shields-fossa-security]: https://app.fossa.com/api/projects/custom%2B48828%2Fgithub.com%2Fblack-desk%2Fcgtproxy.svg?type=shield&issueType=security
[fossa-security]: https://app.fossa.com/projects/custom%2B48828%2Fgithub.com%2Fblack-desk%2Fcgtproxy?ref=badge_shield&issueType=security
[shields-commit-activity]: https://img.shields.io/github/commit-activity/w/black-desk/cgtproxy/master
[shields-contributors]: https://img.shields.io/github/contributors/black-desk/cgtproxy
[shields-release-date]: https://img.shields.io/github/release-date/black-desk/cgtproxy
[shields-commits-since-release]: https://img.shields.io/github/commits-since/black-desk/cgtproxy/latest/master

`cgtproxy` is a transparent proxy **RULE** manager written in go
inspired by [cgproxy].

[cgproxy]: https://github.com/springzfx/cgproxy

`cgtproxy` make it easier to set per-app transparent proxy dynamically.
It will automatically update your nft ruleset according to your configuration,
redirect network traffic in each cgroup to a [target].

[target]: https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html#TARGETS

Currently supported target are:

- DIRECT
- DROP
- TPROXY

## Usage

[Install](./docs/install.md) cgtproxy,
then enable and start the systemd service by:

```bash
# Run the line below if you have old cgtproxy running as systemd service already.
# systemctl daemon-reload
systemctl enable --now cgtproxy.service
```

You could check the nft rules generated by default (empty) configuration by:

```bash
sudo nft list ruleset
```

Write your own configuration file according to the [configuration guide],
place it on /etc/cgtproxy/config.yaml then restart systemd service by:

```bash
systemctl restart cgtproxy.service
```

[configuration guide]: ./docs/configuration.md

## Tips

1. cgproxy has CLI utilities `cgproxy` and `cgnoproxy`
   to temporarily run program with(out) proxy.

   If you use the [example configuration] of `cgtproxy`,
   you can write a bash alias as this:

   ```bash
   alias cgtproxy-direct="systemd-run --user --slice cgtproxy-direct.slice"
   ```

   Then use it like this:

   ```bash
   cgtproxy-direct /some/command/to/run/without/proxy
   ```

   Go check the comments in example configuration
   about the `cgtproxy-direct.slice`, `cgtproxy-drop.slice`
   and `cgtproxy-proxy.slice`.

[example configuration]: ./misc/config/example.yaml

## The way how `cgtproxy` works.

Netfliter can be configured to filter network traffic [by cgroup],
as well as redirect some traffic to a [TPROXY] server.

[by cgroup]: https://www.spinics.net/lists/netfilter/msg60360.html
[TPROXY]: https://www.infradead.org/~mchehab/kernel_docs/networking/tproxy.html

Systemd has a work-in-progress XDG integration [documentation] suggest that
XDG applications should be launched in a systemd managed unit.

[documentation]: https://systemd.io/DESKTOP_ENVIRONMENTS

For example, [telegram] comes from [flatpak]
launched by desktop environment
from the graph session of user whose uid is 1000
should has all its processes running in a cgroup like:

`/user.slice/user-1000.slice/user@1000.service/app.slice/app-flatpak-org.telegram.desktop@12345.service`

[telegram]: https://github.com/telegramdesktop/tdesktop
[flatpak]: https://github.com/flatpak/flatpak

That means the cgroup path of an application instance has a pattern,
which can be match by a regex expression.

`cgtproxy` will listening cgroupfs change with inotify.
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
   3. It create cgroup hierarchy without let systemd known.
      This behavior break the [single-writer rule]
      of design rules of the systemd cgroup API.

      [single-writer rule]: https://systemd.io/CGROUP_DELEGATION#two-key-design-rules

By using `cgtproxy`,
you can have flexible user-level per-app transparent network proxy settings
without any problems above.

## Differences between cgproxy

There are some differences between cgproxy and `cgtproxy`:

- cgproxy using iptables, but `cgtproxy` use nftables.

  Go check differences between iptables and nftables [here][differences].

  [differences]: https://wiki.nftables.org/wiki-nftables/index.php/Main_differences_with_iptables

- cgproxy can only working with exsiting cgroup,
  but `cgtproxy` can update rules dynamically for newly created cgroups;

- cgproxy use BPF to move your process from its original cgroup,
  but `cgtproxy` not;

  cgproxy implement per-app proxy by using BPF to trace the execve syscall.
  If the new executable file of that process matched some pattern,
  cgproxy daemon will put that process into a special hierarchy `proxy.slice`.

  This weird behavior make process escape from the user-level hierarchy,
  which means the systemd resource control settings will not take any effect.

  But `cgtproxy` implement per-app proxy
  by update nftable rules to match newly created cgroups.
  It do not write anything to cgroupv2 filesystem at all.

- cgproxy requires more capability (permission) than `cgtproxy`;

  cgtproxy requires at least CAP_NETWORK_ADMIN and CAP_BPF,
  but cgtproxy require only CAP_NETWORK_ADMIN.

  Check the [systemd service file] for details.

  [systemd service file]: https://github.com/search?q=repo%3Ablack-desk%2Fcgtproxy%20CapabilityBoundingSet&type=code

## TODO

- [ ] ~~optional cgroup monitor implementation listening on D-Bus
      instead of filesystem;~~

  [notify](https://github.com/rjeczalik/notify)
  makes the filesystem monitor much more stable,
  there is no need to implement another monitor for my person usage.

- [ ] DNS hijack for fake-ip;

  - [x] ipv4;

  - [ ] ~~ipv6;~~

    I don't have any ipv6 only device, so I don't need this feature.

- [ ] ~~builtin TPROXY server.~~

  ~~Clash~~
  ~~Clash.Meta~~
  [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo) is good enough for me.

If you need any feature above, PR is welcome.
