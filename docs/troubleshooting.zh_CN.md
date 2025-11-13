<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# 故障排除

[en](./troubleshooting.md) | zh_CN

## `file exists`

```text
Error:
running cgtproxy core: running route manager: add route: file exists
running filesystem watcher: context canceled
```

"file exists"是EEXIST的错误消息，这意味着：

1. 可能有另一个cgtproxy正在运行；

   尝试停止另一个cgtproxy。

2. 之前的cgtproxy没有正常退出，可能被SIG_KILL信号杀死。

   系统重启应该能解决这个问题，如果你不想重启，请按照以下说明操作：

   1. 检查具体发生了什么：

      ```bash
      # 列出cgtproxy创建的路由表中的路由规则。
      # 这里的300是配置中"route-table"的默认值。
      ip route list table 300
      # > local default dev lo scope host

      # 列出cgtproxy创建的nft规则集。
      sudo nft list table inet cgtproxy
      # > table inet cgtproxy {
      # >   set bypass {
      # >           type ipv4_addr
      # >           flags interval
      # > ...

      # 确保cgtproxy没有运行。
      sudo lsof /usr/local/bin/cgtproxy 2>/dev/null
      # 无输出
      ```

   2. 现在你可以简单地通过以下命令删除路由表和nftables规则集：

      ```bash
      sudo ip route del table 300 local default dev lo scope host
      sudo ip -6 route del table 300 local default dev lo scope host
      sudo ip rule del fwmark 3000 lookup 300
      sudo ip -6 rule del fwmark 3000 lookup 300
      sudo nft flush table inet cgtproxy
      sudo nft delete table inet cgtproxy
      ```

## 事件丢失

如果你注意到某些cgroup事件没有被cgtproxy捕获，可能是由于文件系统监视器中的事件丢失。当事件接收器处理事件的速度太慢时就会发生这种情况。

你可能会观察到以下症状：

1. 对于创建事件丢失：
   某些cgroup存在但在nftables中没有对应的规则。

2. 对于删除事件丢失：
   当运行`nft list ruleset`时，你可能会发现nftables中的规则引用了具有inode号码而不是路径的cgroup。
   这是因为当cgroup路径在文件系统中不再存在时，内核只能提供inode号码。

要检查是否遇到事件丢失：

```bash
# 列出目标路径下的所有cgroup
find /sys/fs/cgroup/user.slice -type d

# 检查nft规则
# 如果你在"meta cgroup"表达式中看到inode号码而不是路径，
# 这意味着那些cgroup已被删除，但cgtproxy错过了删除事件
sudo nft list ruleset | grep cgroup
```

要**缓解**这个问题，你可以通过设置`CGTPROXY_MONITOR_BUFFER_SIZE`环境变量来增加事件缓冲区大小。例如：

```bash
# 将缓冲区大小增加到2048（默认为1024）
CGTPROXY_MONITOR_BUFFER_SIZE=2048 cgtproxy
```

你也可以在systemd服务文件中设置此环境变量：

```ini
[Service]
Environment=CGTPROXY_MONITOR_BUFFER_SIZE=2048
```

## DNS 解析未被重定向

如果你发现某些程序（如 `curl`）的 DNS 请求没有被重定向，
这可能是NSS (Name Service Switch) 机制导致的。

当程序使用libc提供的NSS(Name Service Switch)功能进行域名解析时，
会检查`/etc/nsswitch.conf`配置文件中的`hosts`行。
在一些发行版中，这一行在`dns`之前包含一个`resolve`项：

```
hosts: mymachines resolve [!UNAVAIL=return] files myhostname dns
```

这个`resolve`项对应`libnss-resolve.so`（由 systemd 提供的 NSS 插件），
它**直接通过本地 socket 连接到 `systemd-resolved`**，而不是通过网络栈发送DNS查询。
因此，这些DNS请求完全绕过了nftables规则，无法被cgtproxy重定向。

> [!WARNING]
> 以下解决方案需要修改系统配置。除非你理解这样做的含义和潜在后果，否则不要进行此修改。

可以考虑从`/etc/nsswitch.conf`的`hosts`行中移除`resolve`。
这将使程序使用标准的DNS查询通过网络栈发送请求，从而可以被netfilter规则捕获。

目前在绝大多数情况下，这样做的效果与使用`resolve`插件是一样的：
因为启用systemd-resolved的时候，`/etc/resolv.conf`会被systemd-resolved接管
（通常是指向`127.0.0.53`），
程序最终仍然会通过systemd-resolved进行DNS解析，只是会经过网络栈。
