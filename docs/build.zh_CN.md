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

## 编译

你**应当**使用`make`构建cgtproxy。

**不推荐**使用`go build ./cmd/cgtproxy`编译这个项目。

## 调试构建

通过`make GO_TAGS=debug`添加go构建标签`debug`来生成调试构建二进制文件，它具有以下特性：

1. 使错误携带更多信息，如源位置
2. 在我们更新规则集后调用`nft`在日志中打印规则集
3. 向nft规则集添加调试计数器

## 测试

在仓库根目录运行端到端测试：

```bash
make test
make test GO_TAGS=debug
```

CI会测试普通构建和`debug`构建。`make test`会构建带覆盖率统计的cgtproxy、测试辅助程序和覆盖率工具，然后通过公开命令行接口启动cgtproxy，使用实际的TCP/UDP流量验证路由行为。

### 环境要求

- 使用Linux、cgroup
  v2和可用的systemd用户会话。测试默认通过`systemd-run --user -d -P -q`获取委托的cgroup，以普通用户运行。
- 安装满足`go.mod`要求的Go工具链、Make、Bash，以及`iproute2`、`nftables`、`util-linux`、`coreutils`和`procps`提供的命令。
- 系统需要允许非特权用户创建用户命名空间，并支持测试使用的cgroup、挂载、网络和PID命名空间。内核还需支持nftables、TPROXY、IPv4/IPv6、网桥和veth。

流量验证不依赖外部代理或互联网连接；首次构建可能需要下载Go工具链和模块。

如果出现`Operation not permitted`、cgroup不可写或无法连接用户总线等错误，请先检查用户命名空间策略、systemd用户会话和cgroup委托。CI在Ubuntu上还会设置`kernel.apparmor_restrict_unprivileged_userns=0`以允许测试创建用户命名空间。

### 调试与结果

测试使用`--log-file`保存应用日志，未设置`LOG_LEVEL`时默认使用`debug`级别。

通过`E2E_CASE`选择`test/e2e/cases/`下的单个用例目录，例如：

```bash
make test E2E_CASE=local-routing
make test GO_TAGS=debug E2E_CASE=local-routing
```

测试产物默认位于`.test/`。每次运行的日志和服务端收包记录保存在`.test/results.*/<用例名>/`。优先查看`case.log`和`cgtproxy.log`；用例失败时还会尝试保存nftables规则、策略路由和邻居表，便于排查。

测试通过`GOCOVERDIR`收集完整程序的覆盖率，原始数据保存在`.test/coverage.*`，再转换为Codecov可读取的`coverage.txt`。可以用Go工具查看：

```bash
go tool cover -func=coverage.txt
go tool cover -html=coverage.txt
```

可以通过`TEST_BUILD_DIR`和`GO_COVERAGE_PROFILE`覆盖产物目录和覆盖率报告路径。

测试原理和新增用例要求见[开发说明的测试章节](./development.zh_CN.md#测试)。
