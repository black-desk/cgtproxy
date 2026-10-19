<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# 配置指南

[en](./configuration.md) | zh_CN

你可以参考[示例配置](../misc/config/example.yaml)以及[配置字段定义](../internal/cgtproxy/config/config.go)。

## 根 cgroup 和转发流量

路径 `/` 按照 `rules`
的顺序匹配，第一条命中的规则确定根 cgroup 的目标。根目标同时用于本机流量的兜底，以及 Docker
bridge 容器、虚拟机和局域网设备经本机转发的流量。它支持 `direct`、`drop` 和
`tproxy`。没有规则匹配 `/`
时，保持正常转发。已有的本机 cgroup 规则优先于根目标。

例如，在其他规则前添加以下规则，可以保持转发流量直连，同时继续按后续规则配置本机应用：

```yaml
rules:
  - match: "^/$"
    direct: true
  # 其他应用规则……
```

使用 `match: '/.*'`
的兜底规则也会匹配根路径，因此会将转发流量交给该规则指定的目标。转发流量无法按发送进程的 cgroup 匹配；所有转发来源共用根目标，无需
`gateway` 配置项。

PREROUTING 中先跳过连接回复、访问本机地址、非单播流量，以及目的地址命中 `bypass`
的流量，再处理 TCP/UDP。代理目标的 `no-udp` 和 `no-ipv6`
也适用于根目标；配置的 IPv4 DNS 劫持仍可处理 UDP
53。DNS 劫持在 TPROXY 前执行 DNAT，并清除代理标记，以按 DNS 目标地址正常路由。后续报文只有在原始目标端口为 53、且 DNAT 目标与所选代理的 DNS 配置一致时才跳过打标；其他 DNAT 连接仍按 cgroup 或根目标处理。当转发流量的 DNS 被重定向到
`127.0.0.1` 时，需要在入接口启用 `route_localnet`，例如
`net.ipv4.conf.docker0.route_localnet=1`。也可以使用能够从该接口访问的 DNS 地址。

cgtproxy 不修改系统的转发开关、接口设置或其他防火墙规则。直连转发所需的 IP
forwarding、放行规则和必要的 NAT 由网络管理员配置；Docker 通常已配置这些项目。
