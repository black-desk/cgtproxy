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

为了避免破坏nft配置，有很大一部分测试应当在一个网络命名空间中运行。`make test`会创建一个网络命名空间并为一个环境变量赋值来启用这些测试。你可以查看[Makefile](../Makefile)以及[测试源码](../pkg/nftman/nftman_test.go)确认细节。
