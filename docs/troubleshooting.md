# Troubleshooting

---

```
Error:
running cgtproxy core: running route manager: add route: file exists
running filesystem watcher: context canceled
```

The "file exists" is error message of EEXIST, which means that:

1. Maybe there is another cgtproxy running;

   Try to stop another cgtproxy.

2. Previous cgtproxy doesn't exit normally, perhaps killed with SIG_KILL.

   A system reboot should fix this issue, if you don't want to do so read the instructions below:

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
