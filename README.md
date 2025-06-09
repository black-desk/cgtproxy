<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# cgtproxy

[![license][badge-shields-io-license]][license-file]
[![checks][badge-shields-io-checks]][actions]
[![go report card][badge-go-report-card]][go-report-card]
[![codecov][badge-shields-io-codecov]][codecov]
[![commit activity][badge-shields-io-commit-activity]][commits]
[![contributors][badge-shields-io-contributors]][contributors]
[![release date][badge-shields-io-release-date]][releases]
[![commits since release][badge-shields-io-commits-since-release]][commits]

[badge-shields-io-license]: https://img.shields.io/github/license/black-desk/cgtproxy
[license-file]: LICENSE
[badge-shields-io-checks]: https://img.shields.io/github/check-runs/black-desk/cgtproxy/master
[actions]: https://github.com/black-desk/cgtproxy/actions
[badge-go-report-card]: https://goreportcard.com/badge/github.com/black-desk/cgtproxy
[go-report-card]: https://goreportcard.com/report/github.com/black-desk/cgtproxy
[badge-shields-io-codecov]: https://codecov.io/github/black-desk/cgtproxy/graph/badge.svg?token=6TSVGQ4L9X
[codecov]: https://codecov.io/github/black-desk/cgtproxy
[badge-shields-io-commit-activity]: https://img.shields.io/github/commit-activity/w/black-desk/cgtproxy/master
[commits]: https://github.com/black-desk/cgtproxy/commits/master
[badge-shields-io-contributors]: https://img.shields.io/github/contributors/black-desk/cgtproxy
[contributors]: https://github.com/black-desk/cgtproxy/graphs/contributors
[badge-shields-io-release-date]: https://img.shields.io/github/release-date/black-desk/cgtproxy
[releases]: https://github.com/black-desk/cgtproxy/releases
[badge-shields-io-commits-since-release]: https://img.shields.io/github/commits-since/black-desk/cgtproxy/latest/master

`cgtproxy` is a transparent proxy **RULE** manager written in Go,
inspired by [cgproxy].
It makes it easier to set per-app transparent proxy dynamically
by automatically updating your nft ruleset according to your configuration,
redirecting network traffic in each cgroup to a specific [target].

[cgproxy]: https://github.com/springzfx/cgproxy
[target]: https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html#TARGETS

Currently supported targets are:

- DIRECT
- DROP
- TPROXY

## Quick Start

1. [Install](./docs/install.md) cgtproxy

2. Enable and start the systemd service:

   ```bash
   # Run this if you have old cgtproxy running as systemd service
   # systemctl daemon-reload
   systemctl enable --now cgtproxy.service
   ```

   Check the nft rules generated with [the default configuration]:

   ```bash
   sudo nft list ruleset
   ```

3. Create your own configuration:
   - Write your configuration following the [configuration guide]
   - Place it at `/etc/cgtproxy/config.yaml`
   - Restart the service:

     ```bash
     systemctl restart cgtproxy.service
     ```

[the default configuration]: https://pkg.go.dev/github.com/black-desk/cgtproxy/pkg/cgtproxy/config#pkg-constants
[configuration guide]: ./docs/configuration.md

## Usage Tips

You can create a bash function for convenient process management:

```bash
function cgtproxy-exec() {
  local slice="cgtproxy-$1.slice"
  shift 1
  systemd-run --user --slice "$slice" -P "$@"
}
```

Use it like this:

```bash
# Run without proxy
cgtproxy-exec direct /some/command

# Run without network
cgtproxy-exec drop /some/command

# Run with proxy
cgtproxy-exec proxy /some/command
```

Check the [example configuration] for details about
`cgtproxy-direct.slice`, `cgtproxy-drop.slice`,
and `cgtproxy-proxy.slice`.

[example configuration]: ./misc/config/example.yaml

## How It Works

Netfilter can be configured to:

- Filter network traffic [by cgroup]
- Redirect traffic to a [TPROXY] server

[by cgroup]: https://www.spinics.net/lists/netfilter/msg60360.html
[TPROXY]: https://www.infradead.org/~mchehab/kernel_docs/networking/tproxy.html

Systemd's [XDG integration documentation] suggests that
XDG applications should be launched in a systemd managed unit:

> ...
>
> To ensure cross-desktop compatibility and encourage sharing of good practices,
> desktop environments should adhere to the following conventions:
>
> - Application units should follow the scheme
>   `app[-<launcher>]-<ApplicationID>[@<RANDOM>].service` or
>   `app[-<launcher>]-<ApplicationID>-<RANDOM>.scope`[^application-id] e.g:
>
>   - `app-gnome-org.gnome.Evince@12345.service`
>   - `app-flatpak-org.telegram.desktop@12345.service`
>   - `app-KDE-org.kde.okular@12345.service`
>   - `app-org.kde.amarok.service`
>   - `app-org.gnome.Evince-12345.scope`
>
> ...

[^application-id]: <https://specifications.freedesktop.org/desktop-entry-spec/latest/file-naming.html#desktop-file-id>

For example, [Telegram] from [Flatpak] launched by desktop environment
will run in a cgroup like:

```plaintext
/user.slice/user-1000.slice/user@1000.service/app.slice/app-flatpak-org.telegram.desktop@12345.service
```

[XDG integration documentation]: https://systemd.io/DESKTOP_ENVIRONMENTS
[Telegram]: https://github.com/telegramdesktop/tdesktop
[Flatpak]: https://github.com/flatpak/flatpak

This means each application instance's cgroup path follows a pattern
that can be matched by regex.
`cgtproxy` monitors cgroupfs changes with inotify
and updates nftable rules when new cgroup hierarchies are created.

## Why Use cgtproxy?

Common approaches to app-level proxy configuration on Linux have limitations:

1. Environment Variables:
   - Not elegant to configure
   - Some applications ignore them

2. Process-Name-Based Routing (e.g., Clash):
   - Uses [procfs scanning][clash-procfs] for new connections
   - Performance issues with many processes
   - Configuration changes require restart

3. TPROXY with [cgproxy]:
   - Only updates iptables for existing cgroups
   - Uses BPF for new processes
   - Has serious issues:
     - Removes processes from original cgroups
     - Easy unauthorized cgroup escape
     - Breaks systemd's [single-writer rule]

[clash-procfs]: https://github.com/Dreamacro/clash/blob/4d66da2277ddaf41f83bd889b064c0a584f7a8ad/component/process/process_linux.go#L129
[single-writer rule]: https://systemd.io/CGROUP_DELEGATION#two-key-design-rules

`cgtproxy` provides flexible user-level per-app transparent proxy
without these issues.

## Comparison with cgproxy

Key differences:

1. Technology:
   - cgproxy: iptables
   - cgtproxy: nftables ([differences])

2. Cgroup Handling:
   - cgproxy: Only existing cgroups
   - cgtproxy: Dynamic updates for new cgroups

3. Process Management:
   - cgproxy: Uses BPF to move processes
   - cgtproxy: Leaves processes in original cgroups

4. Permissions:
   - cgproxy: Needs CAP_NETWORK_ADMIN and CAP_BPF
   - cgtproxy: Only needs CAP_NETWORK_ADMIN

Check the [systemd service file] for details.

[differences]: https://wiki.nftables.org/wiki-nftables/index.php/Main_differences_with_iptables
[systemd service file]: https://github.com/search?q=repo%3Ablack-desk%2Fcgtproxy%20CapabilityBoundingSet&type=code

## Documentation

Project Documentation:

- [GoDoc][godoc]
- [GitHub Wiki][github-wiki]
- [![DeepWiki][badge-deepwiki]][deepwiki]

[godoc]: https://pkg.go.dev/github.com/black-desk/cgtproxy
[github-wiki]: https://github.com/black-desk/cgtproxy/wiki
[badge-deepwiki]: https://deepwiki.com/badge.svg
[deepwiki]: https://deepwiki.com/black-desk/cgtproxy

Netfilter Documentation:

- [Netfilter/iptables Documentation][netfilter-documentation]

[netfilter-documentation]: https://www.netfilter.org/documentation/index.html

## Development Status

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

---

<picture>
  <source
    media="(prefers-color-scheme: dark)"
    srcset="
      https://api.star-history.com/svg?repos=black-desk/cgtproxy&type=Date&theme=dark
    "
  />
  <source
    media="(prefers-color-scheme: light)"
    srcset="
      https://api.star-history.com/svg?repos=black-desk/cgtproxy&type=Date
    "
  />
  <img
    alt="Star History Chart"
    src="https://api.star-history.com/svg?repos=black-desk/cgtproxy&type=Date"
  />
</picture>
