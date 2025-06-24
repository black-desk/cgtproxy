<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# cgtproxy

[![checks][badge-shields-io-checks]][actions]
[![commit activity][badge-shields-io-commit-activity]][commits]
[![contributors][badge-shields-io-contributors]][contributors]
[![release date][badge-shields-io-release-date]][releases]
![commits since release][badge-shields-io-commits-since-release]
[![codecov][badge-shields-io-codecov]][codecov]
[![go report card][badge-go-report-card]][go-report-card]

[badge-shields-io-checks]:
  https://img.shields.io/github/check-runs/black-desk/cgtproxy/master
[actions]: https://github.com/black-desk/cgtproxy/actions
[badge-shields-io-commit-activity]:
  https://img.shields.io/github/commit-activity/w/black-desk/cgtproxy/master
[commits]: https://github.com/black-desk/cgtproxy/commits/master
[badge-shields-io-contributors]:
  https://img.shields.io/github/contributors/black-desk/cgtproxy
[contributors]: https://github.com/black-desk/cgtproxy/graphs/contributors
[badge-shields-io-release-date]:
  https://img.shields.io/github/release-date/black-desk/cgtproxy
[releases]: https://github.com/black-desk/cgtproxy/releases
[badge-shields-io-commits-since-release]:
  https://img.shields.io/github/commits-since/black-desk/cgtproxy/latest
[badge-shields-io-codecov]:
  https://codecov.io/github/black-desk/cgtproxy/graph/badge.svg?token=6TSVGQ4L9X
[codecov]: https://codecov.io/github/black-desk/cgtproxy
[badge-go-report-card]:
  https://goreportcard.com/badge/github.com/black-desk/cgtproxy
[go-report-card]: https://goreportcard.com/report/github.com/black-desk/cgtproxy

en | [zh_CN](README.zh_CN.md)

> [!WARNING]
>
> This English README is translated from the Chinese version using AI and may
> contain errors.

`cgtproxy` is a transparent proxy **rule** manager written in Go, inspired by
the [cgproxy] project.

This project automatically updates [nft] rules based on configuration files to
redirect network traffic from each [cgroup] to specific targets, enabling
dynamic transparent proxy rules at the application level.

[nft]: https://www.netfilter.org/projects/nftables/manpage.html
[cgproxy]: https://github.com/springzfx/cgproxy
[cgroup]: https://man7.org/linux/man-pages/man7/cgroups.7.html

Currently supported targets include:

- `DIRECT` (direct connection)
- `DROP` (drop packets)
- [`TPROXY`][TPROXY]

[TPROXY]: https://www.infradead.org/~mchehab/kernel_docs/networking/tproxy.html

## Usage

1. [Install](./docs/install.md) `cgtproxy`

2. Enable and start the systemd service:

   ```bash
   systemctl daemon-reload
   systemctl enable --now cgtproxy.service
   ```

   Check the generated nft rules with the [default configuration]:

   ```bash
   sudo nft list ruleset
   ```

3. Create your own configuration:
   - Write a configuration following the [configuration guide]
   - Place the configuration file at `/etc/cgtproxy/config.yaml`
   - Restart the service:

     ```bash
     systemctl restart cgtproxy.service
     ```

[default configuration]:
  https://pkg.go.dev/github.com/black-desk/cgtproxy/pkg/cgtproxy/config#pkg-constants
[configuration guide]: ./docs/configuration.md

## Tips

You can create a bash function to manage processes conveniently:

```bash
function cgtproxy-exec() {
  local slice="cgtproxy-$1.slice"
  shift 1
  systemd-run --user --slice "$slice" -P "$@"
}
```

When you use this function like this:

```bash
# Run without proxy
cgtproxy-exec direct /some/command

# Run with network disabled
cgtproxy-exec drop /some/command

# Run with proxy
cgtproxy-exec proxy /some/command
```

The function uses `systemd-run` to run commands in the `cgtproxy-direct`,
`cgtproxy-drop`, and `cgtproxy-proxy` slices.

In the [example configuration], we:

- Leave traffic from cgroup `cgtproxy-direct.slice` unchanged;
- Drop traffic from cgroup `cgtproxy-drop.slice`;
- Transparently proxy traffic from cgroup `cgtproxy-proxy.slice`.

[example configuration]: ./misc/config/example.yaml

## How It Works

Netfilter can filter network traffic [by cgroup] and redirect traffic to
[TPROXY] servers.

[by cgroup]: https://www.spinics.net/lists/netfilter/msg60360.html

The [systemd documentation for desktop environments] suggests that desktop
environments and other launchers should start applications in systemd-managed
units:

[systemd documentation for desktop environments]:
  https://systemd.io/DESKTOP_ENVIRONMENTS/

> ...
>
> To ensure cross-desktop compatibility and encourage sharing of good practices,
> desktop environments should adhere to the following conventions:
>
> - Application units should be named
>   `app[-<launcher>]-<ApplicationID>[@<RANDOM>].service` or
>   `app[-<launcher>]-<ApplicationID>-<RANDOM>.scope`. For example:
>   - `app-gnome-org.gnome.Evince@12345.service`
>   - `app-flatpak-org.telegram.desktop@12345.service`
>   - `app-KDE-org.kde.okular@12345.service`
>   - `app-org.kde.amarok.service`
>   - `app-org.gnome.Evince-12345.scope`
>
> ...

While this documentation doesn't directly specify what "Application ID" should
contain, all major Linux desktop environments currently use "[desktop file ID]"
when launching applications.

[desktop file ID]:
  https://specifications.freedesktop.org/desktop-entry-spec/latest/file-naming.html#desktop-file-id

For example, [Telegram] from [Flatpak] launched by a desktop environment will
run in a cgroup similar to:

[Telegram]: https://github.com/telegramdesktop/tdesktop
[Flatpak]: https://github.com/flatpak/flatpak

```plaintext
/user.slice/user-1000.slice/user@1000.service/app.slice/app-flatpak-org.telegram.desktop@12345.service
```

This means that the [cgroup] path of each application instance follows a pattern
that can be matched using regular expressions.

Based on this, `cgtproxy` monitors [cgroupfs][cgroup] changes through [inotify],
and uses regex-based configuration files to update [nft] rules according to
configuration when new [cgroup] hierarchies are created.

[inotify]: https://man7.org/linux/man-pages/man7/inotify.7.html

## Advantages

Common methods for application-level proxy configuration on Linux have
limitations:

1. Environment variables:
   - No simple way to configure at the application level.
   - Some applications ignore environment variables.

2. Filtering traffic by binary path (like Clash):

   Clash [scans procfs][clash-procfs] when applications create new sockets to
   determine which binary actually initiated the connection, then decides how to
   route it.

   [clash-procfs]:
     https://github.com/Dreamacro/clash/blob/4d66da2277ddaf41f83bd889b064c0a584f7a8ad/component/process/process_linux.go#L129

   This approach has the following issues:
   - Performance issues when there are many processes
   - Applications written with scripts (like those using [pyqt]) cannot be
     correctly identified
   - Cannot correctly identify when applications connect to the network through
     command-line tools

   [pyqt]: https://doc.qt.io/qtforpython-6/

3. Using [cgproxy]:

   This approach has serious security issues:
   - cgproxy moves processes out of their original cgroups
   - Risk of unauthorized user-level processes escaping to system-level cgroups
   - This approach violates systemd's [single-writer rule]

   [single-writer rule]:
     https://systemd.io/CGROUP_DELEGATION#two-key-design-rules

`cgtproxy` does not have these problems.

## Differences from cgproxy

1. `cgproxy` uses [iptables], while `cgtproxy` uses [nftables].

   You can check the differences between [iptables] and [nftables]
   [here][nftables_differences_with_iptables].

   [iptables]: https://linux.die.net/man/8/iptables
   [nftables]: https://wiki.archlinux.org/title/Nftables
   [nftables_differences_with_iptables]:
     https://wiki.nftables.org/wiki-nftables/index.php/Main_differences_with_iptables

2. `cgproxy` predefines several [cgroup]s and creates routing rules for them;
   while `cgtproxy` doesn't create [cgroup]s, but dynamically updates routing
   rules when [cgroup]s appear.

3. `cgproxy` uses eBPF to hook the execve system call, determining executable
   file paths when all applications execute and moving processes to predefined
   [cgroup]s; while `cgtproxy` keeps processes in their original [cgroup]s.

4. `cgproxy` requires `CAP_SYS_ADMIN`, `CAP_NETWORK_ADMIN`, and `CAP_BPF`; while
   `cgtproxy` only requires `CAP_NETWORK_ADMIN`. See the [systemd service file]
   for details.

[systemd service file]:
  https://github.com/search?q=repo%3Ablack-desk%2Fcgtproxy%20CapabilityBoundingSet&type=code

## Documentation

Project documentation:

- [GoDoc][godoc]
- [GitHub Wiki][github-wiki]
- [![DeepWiki][badge-deepwiki]][deepwiki]

[godoc]: https://pkg.go.dev/github.com/black-desk/cgtproxy
[github-wiki]: https://github.com/black-desk/cgtproxy/wiki
[badge-deepwiki]: https://deepwiki.com/badge.svg
[deepwiki]: https://deepwiki.com/black-desk/cgtproxy

Netfilter documentation:

- [Netfilter/iptables documentation][netfilter-documentation]

[netfilter-documentation]: https://www.netfilter.org/documentation/index.html

## Development Status

- [ ] ~~Optional cgroup monitoring implementation that listens to D-Bus instead
      of filesystem;~~

  [notify](https://github.com/rjeczalik/notify) makes filesystem monitoring more
  stable. For my personal use, there's no need to implement another monitoring
  mechanism.

- [ ] DNS hijacking for fake-ip;
  - [x] ipv4;

  - [ ] ~~ipv6;~~

    I don't have any IPv6-only devices, so I don't need this feature and cannot
    verify it.

- [ ] ~~Built-in TPROXY server.~~

  ~~Clash~~ ~~Clash.Meta~~
  [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo) is good enough for me.

If you need any of the above features, pull requests are welcome.

## License

Unless otherwise stated, the code in this project is licensed under the GNU
General Public License version 3 or any later version, while documentation,
configuration files, and scripts used in the development and maintenance process
are licensed under the MIT License.

This project complies with the [REUSE Specification].

You can use [reuse-tool](https://github.com/fsfe/reuse-tool) to generate an SPDX
list for this project:

```bash
reuse spdx
```

[REUSE Specification]: https://reuse.software/spec-3.3/

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
