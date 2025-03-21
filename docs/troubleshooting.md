# Troubleshooting

## File exists

```
Error:
running cgtproxy core: running route manager: add route: file exists
running filesystem watcher: context canceled
```

The "file exists" is error message of EEXIST, which means that:

1. Maybe there is another cgtproxy running;

   Try to stop another cgtproxy.

2. Previous cgtproxy doesn't exit normally, perhaps killed with SIG_KILL.

   A system reboot should fix this issue,
   if you don't want to do so read the instructions below:

   1. Check what is exactly going on:

      ```bash
      # List route rules in the table cgtproxy created.
      # 300 is the default value of "route-table" in configuration.
      ip route list table 300
      # > local default dev lo scope host

      # List nft ruleset created by cgtproxy.
      sudo nft list table inet cgtproxy
      # > table inet cgtproxy {
      # >   set bypass {
      # >           type ipv4_addr
      # >           flags interval
      # > ...

      # Make sure cgtproxy is not running.
      sudo lsof /usr/local/bin/cgtproxy 2>/dev/null
      # No output
      ```

   2. Now you can just simply remove the route table and nftable ruleset by:

      ```bash
      sudo ip route del table 300 local default dev lo scope host
      sudo ip -6 route del table 300 local default dev lo scope host
      sudo ip rule del fwmark 3000 lookup 300
      sudo ip -6 rule del fwmark 3000 lookup 300
      sudo nft flush table inet cgtproxy
      sudo nft delete table inet cgtproxy
      ```

## Event Loss

If you notice that some cgroup events are not being captured by cgtproxy,
it might be due to event dropping in the filesystem monitor.
This can happen when the event receiver is too slow to process events.

You may observe the following symptoms:

1. For creation events loss:
   Some cgroups exist but have no corresponding rules in nftables.

2. For deletion events loss:
   You may find rules in nftables referencing cgroups with inode numbers
   instead of paths when running `nft list ruleset`.
   This happens because the kernel can only provide the inode number
   when the cgroup path no longer exists in the filesystem.

To check if you are experiencing event loss:

```bash
# List all cgroups under your target path
find /sys/fs/cgroup/user.slice -type d

# Check nft rules
# If you see inode numbers in "meta cgroup" expressions instead of paths,
# it means those cgroups have been deleted but cgtproxy missed the deletion events
sudo nft list ruleset | grep cgroup
```

To mitigate this issue,
you can increase the event buffer size by setting
the `CGTPROXY_MONITOR_BUFFER_SIZE` environment variable.
For example:

```bash
# Increase buffer size to 2048 (default is 1024)
CGTPROXY_MONITOR_BUFFER_SIZE=2048 cgtproxy
```

You can also set this in the systemd service file
by adding the environment variable:

```ini
[Service]
Environment=CGTPROXY_MONITOR_BUFFER_SIZE=2048
```

Note that a larger buffer size will consume more memory
but can handle more events in a short period.
If you still experience event loss after increasing the buffer size,
you might need to:

1. Further increase the buffer size
2. Check if your system is under heavy load
3. Consider reducing the rate of cgroup creation/deletion if possible
