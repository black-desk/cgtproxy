<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# 配置指南

[en](./configuration.md) | zh_CN

你可以参考[示例配置](../misc/config/example.yaml)以及[GoDoc][godoc]。

[godoc]: https://pkg.go.dev/github.com/black-desk/cgtproxy

## 网关（Gateway）

除了 `TPROXY` 目标之外，cgtproxy 还可以把某个 cgroup 的流量路由到指定的下一跳
网关，由网关自行做透明代理：

```yaml
gateways:
  home-router:
    ip: 192.168.1.1
    dev: eth0
    mark: 4000

rules:
  - match: /system.slice/foo.service
    gateway: home-router
```

每个网关需要配置：

- `ip`：下一跳网关的地址（IPv4 或 IPv6）。只有该地址所属的 family 会被路由到
  网关；命中网关规则的 cgroup 的另一 family 流量会走默认路由直接发出。
- `dev`：到达网关所用的网络接口。网关会被视为 on-link，因此不需要递归查找网关
  的下一跳。
- `mark`：用于选择该网关路由表的 fwmark。它必须与所有 tproxy 和其他网关的
  mark 全局唯一。

对于每个网关，cgtproxy 会添加：

- 一个 nft 链 `<名字>-MARK`，用于设置 fwmark；
- `ip rule add fwmark <mark> lookup <table>`；
- `ip route add default via <ip> dev <dev> table <table> onlink`。

路由表号会自动分配：按网关名字排序后，第一个网关使用 `route-table + 1`，
第二个使用 `route-table + 2`，依此类推。

> 注意：网关的路由表里只有经由网关的默认路由，所以被网关规则命中的 cgroup
> 发往回环或本机地址的流量必须放进 `bypass`，否则也会被送到网关。
