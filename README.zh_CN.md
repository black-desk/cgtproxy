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

[en](README.md) | zh_CN

`cgtproxy`是一个受[cgproxy]项目启发，用Go语言编写的透明代理**规则**管理器。

该项目通过根据配置文件自动更新[nft]规则，将每个[cgroup]中的网络流量重定向到特定的目标，可以以应用为粒度设置动态的透明代理规则。

[nft]: https://www.netfilter.org/projects/nftables/manpage.html
[cgproxy]: https://github.com/springzfx/cgproxy
[cgroup]: https://man7.org/linux/man-pages/man7/cgroups.7.html

目前支持的目标包括：

- `DIRECT`（直连）
- `DROP`（丢弃）
- [`TPROXY`][TPROXY]

[TPROXY]: https://www.infradead.org/~mchehab/kernel_docs/networking/tproxy.html

## 使用

1. [安装](./docs/install.zh_CN.md)`cgtproxy`

2. 启用并启动systemd服务：

   ```bash
   systemctl daemon-reload
   systemctl enable --now cgtproxy.service
   ```

   使用[默认配置]检查生成的nft规则：

   ```bash
   sudo nft list ruleset
   ```

3. 创建您自己的配置：
   - 根据[配置指南]编写配置
   - 将配置文件放置在 `/etc/cgtproxy/config.yaml`
   - 重启服务：

     ```bash
     systemctl restart cgtproxy.service
     ```

[默认配置]:
  https://pkg.go.dev/github.com/black-desk/cgtproxy/pkg/cgtproxy/config#pkg-constants
[配置指南]: ./docs/configuration.zh_CN.md

## 技巧

您可以创建一个 bash 函数来方便地管理进程：

```bash
function cgtproxy-exec() {
  local slice="cgtproxy-$1.slice"
  shift 1
  systemd-run --user --slice "$slice" -P "$@"
}
```

当您这样使用这个函数时：

```bash
# 不使用代理运行
cgtproxy-exec direct /some/command

# 禁用网络运行
cgtproxy-exec drop /some/command

# 使用代理运行
cgtproxy-exec proxy /some/command
```

该函数使用`systemd-run`在`cgtproxy-direct`、`cgtproxy-drop`和`cgtproxy-proxy`这三个`slice`中运行命令。

在[示例配置]中，我们：

- 不修改来自 cgroup `cgtproxy-direct.slice`的流量;
- 丢弃来自 cgroup `cgtproxy-drop.slice`的流量；
- 透明地代理了来自 cgroup `cgtproxy-proxy.slice`的流量。

[示例配置]: ./misc/config/example.yaml

## 工作原理

Netfilter 可以[按cgroup]过滤网络流量，并将流量重定向到[TPROXY]服务器。

[按cgroup]: https://www.spinics.net/lists/netfilter/msg60360.html

而[systemd的桌面环境文档]建议桌面环境等启动器应该在 systemd 管理的单元中启动应用程序：

[systemd的桌面环境文档]: https://systemd.io/DESKTOP_ENVIRONMENTS/

> ...
>
> 为了确保跨桌面兼容性并鼓励分享良好实践，桌面环境应该遵循以下约定：
>
> - 应用程序单元的名称应当形如`app[-<启动器>]-<应用ID>[@<随机内容>].service`或`app[-<启动器名称>]-<应用ID>-<随机内容>.scope`。例如：
>   - `app-gnome-org.gnome.Evince@12345.service`
>   - `app-flatpak-org.telegram.desktop@12345.service`
>   - `app-KDE-org.kde.okular@12345.service`
>   - `app-org.kde.amarok.service`
>   - `app-org.gnome.Evince-12345.scope`
>
> ...

该文档并未直接规定“应用 ID”的内容，但目前各 Linux 桌面启动应用时均使用“[桌面文件ID]”。

[桌面文件ID]:
  https://specifications.freedesktop.org/desktop-entry-spec/latest/file-naming.html#desktop-file-id

例如，由桌面环境启动的来自[Flatpak]的[Telegram]将在类似以下的 cgroup 中运行：

[Telegram]: https://github.com/telegramdesktop/tdesktop
[Flatpak]: https://github.com/flatpak/flatpak

```plaintext
/user.slice/user-1000.slice/user@1000.service/app.slice/app-flatpak-org.telegram.desktop@12345.service
```

这意味着每个应用程序实例的[cgroup]路径都遵循一个可以通过正则表达式匹配的模式。

在这个基础上，`cgtproxy`通过[inotify]监控[cgroupfs][cgroup]的变化，使用基于正则表达式的配置文件，在创建新的[cgroup]层次结构时按配置更新[nft]规则。

[inotify]: https://man7.org/linux/man-pages/man7/inotify.7.html

## 优点

在 Linux 上进行应用级代理配置的常见方法都有局限性：

1. 环境变量：
   - 没有简单的方法可以在应用层面配置。
   - 某些应用程序会忽略环境变量。

2. 通过二进制路径过滤流量（如Clash）：

   Clash会在有应用程序创建了新套接字时[扫描procfs][clash-procfs]，来确定该链接实际上是由哪个二进制发起的，然后决定如何路由。

   [clash-procfs]:
     https://github.com/Dreamacro/clash/blob/4d66da2277ddaf41f83bd889b064c0a584f7a8ad/component/process/process_linux.go#L129

   这个方案有以下问题：
   - 进程较多时可能会存在性能问题
   - 通过脚本编写的程序（如使用[pyqt]的编写的应用）无法被正确判断
   - 应用程序通过运行命令行工具联网时无法正确判断

   [pyqt]: https://doc.qt.io/qtforpython-6/

3. 使用[cgproxy]：

   该方案存在严重的安全问题：
   - cgproxy 会将进程从原始cgroup中移出
   - 存在未授权的用户级进程逃逸到系统级cgroup

   并且该方案破坏了systemd的[单一写入者规则]。

   [单一写入者规则]: https://systemd.io/CGROUP_DELEGATION#two-key-design-rules

`cgtproxy`不存在这些问题。

## 与cgproxy的区别

1. `cgproxy`使用[iptables]，而`cgtproxy`使用[nftables]。

   您可以在[nftables的wiki上][nftables_differences_with_iptables]查看[iptables]和[nftables]的区别。

   [iptables]: https://linux.die.net/man/8/iptables
   [nftables]: https://wiki.archlinux.org/title/Nftables
   [nftables_differences_with_iptables]:
     https://wiki.nftables.org/wiki-nftables/index.php/Main_differences_with_iptables

2. `cgproxy`预定义了数个[cgroup]，并为其创建路由规则；而`cgtproxy`不创建[cgroup]，当[cgroup]出现时动态更新路由规则。

3. `cgproxy`使用eBPF，hook了execve系统调用，在所有应用程序执行时判断其可执行文件路径，将进程移动到预定义的[cgroup]中；而`cgtproxy`将进程保留在原始[cgroup]中。

4. `cgproxy`需要`CAP_SYS_ADMIN`、`CAP_NETWORK_ADMIN`和`CAP_BPF`；而`cgtproxy`只需要`CAP_NETWORK_ADMIN`。详情请查看[systemd服务文件]。

[systemd服务文件]:
  https://github.com/search?q=repo%3Ablack-desk%2Fcgtproxy%20CapabilityBoundingSet&type=code

## 文档

项目文档：

- [GoDoc][godoc]
- [GitHub Wiki][github-wiki]
- [![DeepWiki][badge-deepwiki]][deepwiki]

[godoc]: https://pkg.go.dev/github.com/black-desk/cgtproxy
[github-wiki]: https://github.com/black-desk/cgtproxy/wiki
[badge-deepwiki]: https://deepwiki.com/badge.svg
[deepwiki]: https://deepwiki.com/black-desk/cgtproxy

Netfilter 文档：

- [Netfilter/iptables文档][netfilter-documentation]

[netfilter-documentation]: https://www.netfilter.org/documentation/index.html

## 开发状态

- [ ] ~~可选的cgroup监控实现，监听D-Bus而不是文件系统；~~

  [notify](https://github.com/rjeczalik/notify)使文件系统监控更加稳定，对于我个人使用来说，已经没有必要实现另一种监控机制了。

- [ ] 为fake-ip劫持DNS；
  - [x] ipv4；
  - [ ] ~~ipv6；~~

    我没有任何仅支持ipv6的设备，不需要也无法验证这个功能。

- [ ] ~~内置TPROXY服务器。~~

  ~~Clash~~ ~~Clash.Meta~~
  [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo)对我来说已经足够好了。

如果您需要上述任何功能，欢迎提交拉取请求。

## 许可证

除非另有说明，本项目的代码在GNU通用公共许可证第3版或任何更高版本下开源，而文档、配置文件和开发维护过程中使用的脚本在MIT许可证下开源。

本项目符合[REUSE规范]。

您可以使用[reuse-tool](https://github.com/fsfe/reuse-tool)为本项目生成SPDX列表：

```bash
reuse spdx
```

[REUSE规范]: https://reuse.software/spec-3.3/

---

<!-- markdownlint-disable -->
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
