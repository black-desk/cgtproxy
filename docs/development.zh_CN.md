<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# 开发说明

[en](./development.md) | zh_CN

## 项目结构

本项目使用[wire]来实践[依赖注入]。你需要查看[wire.go]文件来了解这个应用程序是如何构建的。

[wire]: https://github.com/google/wire
[依赖注入]: https://en.wikipedia.org/wiki/Dependency_injection
[wire.go]: ../internal/cmd/wire.go

所有实现包位于
`internal/`，仅供 cgtproxy 自身使用。内部接口用于组织组件，不作为公开的库 API。

cgtproxy的基本工作流程：

```text
文件系统通知器 [github.com/rjeczalik/notify]
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

- 当标准错误是终端时，日志写入标准错误；否则，日志会被发送到journald；

- 你可以使用环境变量`LOG_LEVEL`控制日志级别，可选值为：`debug`、`info`、`warn`、`error`、`dpanic`、`panic`、`fatal`；

- 有部分信息结构化地被打印在字段中，你可以使用[fmtjournal]来查看所有日志字段。

[fmtjournal]: https://github.com/black-desk/fmtjournal

## 构建标签

通过`make GO_TAGS=debug`添加go构建标签`debug`来生成调试构建二进制文件，它具有以下特性：

1. 使错误携带更多信息，如源位置
2. 在我们更新规则集后调用`nft`在日志中打印规则集
3. 向nft规则集添加调试计数器
