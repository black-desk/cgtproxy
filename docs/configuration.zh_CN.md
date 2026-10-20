<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# 配置指南

[en](./configuration.md) | zh_CN

你可以参考[示例配置](../misc/config/example.yaml)以及[配置字段定义](../internal/cgtproxy/config/config.go)。

## 根 cgroup

路径 `/` 按照 `rules`
的顺序匹配，第一条命中的规则确定根 cgroup 的目标。根目标用于本机流量的兜底，支持
`direct`、`drop` 和 `tproxy`。已有的本机 cgroup 规则优先于根目标。

## 网关模式

网关模式用于处理由本机路由的流量，例如将本机作为局域网设备的路由器、为虚拟机转发流量，或接管 Docker
bridge 容器的出站流量。

顶层配置项 `gateway` 默认是 `false`。省略它或显式设置为 `false`
时，经本机转发的流量保持直连，即使根目标为代理或丢弃，本机流量仍按原有规则处理。设置
`gateway: true` 后，经本机转发的流量会共用根 cgroup 的目标。例如：

```yaml
gateway: true
rules:
  - match: "^/$"
    tproxy: clash-meta
  # 其他应用规则……
```

使用 `match: '/.*'`
的兜底规则也会匹配根路径，因此在启用网关模式后会将转发流量交给该规则指定的目标。没有规则匹配
`/` 时，保持正常转发；也可以通过显式的 `match: '^/$'`
规则将根目标设为直连或丢弃。转发流量无法按发送进程的 cgroup 匹配，所有转发来源共用根目标。
`gateway` 是顶层开关，不属于 `tproxies` 中某个代理的配置。

启用网关模式时，PREROUTING 中先跳过连接回复、访问本机地址、非单播流量，以及目的地址命中
`bypass` 的流量，再处理 TCP/UDP。代理目标的 `no-udp` 和 `no-ipv6`
也适用于根目标；配置的 IPv4 DNS 劫持仍可处理 UDP
53。DNS 劫持在 TPROXY 前执行 DNAT，并清除代理标记，以按 DNS 目标地址正常路由。后续报文只有在原始目标端口为 53、且 DNAT 目标与所选代理的 DNS 配置一致时才跳过打标；其他 DNAT 连接仍按 cgroup 或根目标处理。

cgtproxy 不修改系统的转发开关、接口设置或其他防火墙规则。直连转发所需的 IP
forwarding、放行规则和必要的 NAT 由网络管理员配置；Docker 通常已配置这些项目。

如果容器在启用网关模式后出现 DNS 超时，参见[故障排查](./troubleshooting.zh_CN.md#docker-容器在启用网关模式后-dns-超时)。
