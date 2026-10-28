<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# 开发说明

[en](./development.md) | zh_CN

## 编译与测试

编译方法、调试构建、测试环境要求、运行命令、日志和覆盖率说明见[构建指南](./build.zh_CN.md)。

## 项目结构

本项目使用[wire]来实践[依赖注入]。你需要查看[wire.go]文件来了解这个应用程序是如何构建的。

[wire]: https://github.com/google/wire
[依赖注入]: https://en.wikipedia.org/wiki/Dependency_injection
[wire.go]: ../internal/cmd/wire.go

所有实现包位于`internal/`，仅供cgtproxy自身使用。内部接口用于组织组件，不作为公开的库API。

cgtproxy的基本工作流程：

```text
文件系统通知器 [github.com/fsnotify/fsnotify]
|
| 新的inotify事件
v
cgroup监视器 [github.com/black-desk/cgtproxy/internal/cgfsmon]
|
| cgroup事件
v
路由管理器 [github.com/black-desk/cgtproxy/internal/routeman]
|
| 更新nft，使用nftman [github.com/black-desk/cgtproxy/internal/nftman]
v
内核中的netfilter框架
```

## 更新NFTables规则

与用C语言编写的`nft`用户空间工具不同，Google的golang
nftables实现不能像`nft -f ...`那样执行nft脚本，这使得我们必须找出`nft`向netlink套接字写入的底层表达式。

参考该golang包作者的一个[评论]，我们可以使用`nft --debug all -f ...`来检查`nft`中发生了什么。

[评论]: https://github.com/google/nftables/issues/5#issuecomment-451373151

我建议使用`nft --debug netlink -f ...`来检查写入netlink套接字的表达式，这有助于你找出应该使用`github.com/google/nftables/expr`中的哪个结构。

## 日志记录

`check config`默认启用日志，与主命令使用相同的`--log-file`和`LOG_LEVEL`设置。

- 使用`--log-file /path/to/cgtproxy.log`将JSON日志直接追加到文件，替代stderr和journald输出。新文件的权限为`0600`，父目录必须已存在。

- 当标准错误是终端时，日志写入标准错误；否则，日志会被发送到journald；

- 你可以使用环境变量`LOG_LEVEL`控制日志级别，可选值为：`debug`、`info`、`warn`、`error`、`dpanic`、`panic`、`fatal`；

## 测试

测试运行方法见[构建指南的测试章节](./build.zh_CN.md#测试)。

### 测试原理

每个[端到端用例](../test/e2e/cases)都在独立的用户、挂载、网络、PID和cgroup命名空间以及委托的cgroup中运行。测试脚本在其中创建网络拓扑、代理服务器和nftables规则。

测试网络包含一个模拟宿主机、通过网桥连接的客户端，以及单独的原站网络命名空间。测试代理返回服务身份、原始目的地址和请求内容，客户端据此区分代理接管和直连。测试还验证cgroup动态变化和cgtproxy正常退出后的nftables表、策略规则及路由清理。

测试通过SIGTERM正常停止cgtproxy，让退出清理和覆盖率写入都能完成。

### 新增用例

本项目不做单元测试。

新增端到端测试用例时，在`test/e2e/cases/`下创建目录，提供完整的`config.yaml`和`test.sh`，测试运行器会自动发现该用例。公共网络搭建和`start_cgtproxy`、`probe`、`wait_until`等辅助函数见[common.sh](../test/e2e/common.sh)，测试客户端和透明代理实现见[fixture](../test/e2e/fixture/main.go)。用例无需导入内部包或模拟组件接口。

新增或修复路由行为时，使用真实流量验证预期结果，并为修复的问题添加回归场景。现有用例覆盖配置校验、本地cgroup路由与动态更新、网关转发、DNS劫持、协议和地址排除，以及退出时的规则清理。提交前运行普通构建和`debug`构建的测试；如有无法运行的检查，请在提交说明或PR中记录原因和已完成的验证。
