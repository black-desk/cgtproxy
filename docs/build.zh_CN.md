<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# 构建指南

本文中的关键词**必须**、**禁止**、**必要的**、**应当**、**不应**、**推荐的**、**允许**以及**可选的**的解释见于[RFC
2119][rfc-2119]中的描述。

这些关键词与原文中的英语词汇的对应关系如下表所示：

| 中文       | 英语        |
| ---------- | ----------- |
| **必须**   | MUST        |
| **禁止**   | MUST NOT    |
| **必要的** | REQUIRED    |
| **应当**   | SHALL       |
| **不应**   | SHALL NOT   |
| **推荐的** | RECOMMENDED |
| **允许**   | MAY         |
| **可选的** | OPTIONAL    |

[rfc-2119]: https://datatracker.ietf.org/doc/html/rfc2119

---

你**应当**使用`make`构建cgtproxy。

**不推荐**使用`go build ./cmd/cgtproxy`编译这个项目。

## 测试

`make test` 构建带覆盖率的完整 cgtproxy 可执行程序，并运行
[端到端场景](../test/e2e/cases)。每个场景包含 `config.yaml` 和
`test.sh`，在独立的用户、挂载、网络、PID 和 cgroup 命名空间中执行。每个场景分别测试复用和不复用 netlink 套接字。

运行环境需要 Linux、cgroup
v2、可用的 systemd 用户会话、Go、Bash、iproute2、nftables、util-linux 和 procps。`systemd-run --user`
提供可写的委托 cgroup；系统须允许非特权用户命名空间。CI 会配置所需的用户命名空间权限。

```sh
make test
make test E2E_CASE=gateway-proxy
make test GO_TAGS=debug
```

测试网络包含一个模拟宿主机、通过网桥连接的客户端，以及单独的原站网络命名空间。测试代理返回服务身份、原始目的地址和请求内容，客户端据此区分代理接管和直连；丢弃测试同时检查超时或本地防火墙拒绝，以及服务端没有收到报文。测试还验证 cgroup 动态变化和 cgtproxy 正常退出后的 nft、策略规则及路由清理。

日志、服务端收包记录和失败时的网络快照保存在 `.test/results.*`，原始覆盖率保存在
`.test/coverage.*`。测试通过 `GOCOVERDIR`
收集完整程序的覆盖率，并转换为 Codecov 可读取的 `coverage.txt`。可用
`GO_COVERAGE_PROFILE`
指定输出路径。测试通过 SIGTERM 正常停止 cgtproxy，让退出清理和覆盖率写入都能完成。

新增场景时，在 `test/e2e/cases/`
创建目录并添加配置与脚本；公共网络搭建和辅助函数见
[common.sh](../test/e2e/common.sh)，测试客户端和透明代理实现见
[fixture](../test/e2e/fixture/main.go)。场景无需导入内部包或模拟组件接口。
