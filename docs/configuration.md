# Configuration

## Configuration File

The configuration file is in YAML format.
For a complete example, check [example configuration](../misc/config/example.yaml).

The configuration file consists of the following sections:

1. `version`: Required. Must be "1".

2. `cgroup-root`:
   Required. The root path of cgroup v2 filesystem.
   Use "AUTO" to automatically detect.

3. `bypass`:
   Optional. A list of IP addresses or CIDR ranges that should not be proxied.
   Traffic to these destinations will not be touched.

4. `tproxies`:
   Required. A map of TPROXY server configurations.
   Each TPROXY server has the following options:
   - `port`: Required. The port number for the TPROXY server.
   - `mark`: Required. The firewall mark for identifying TPROXY traffic.
   - `no-udp`: Optional. Set to true to disable UDP support.
   - `no-ipv6`: Optional. Set to true to disable IPv6 support.
   - `dns-hijack`: Optional. Configuration for DNS request hijacking.

5. `rules`:
   Required. A list of rules that determine how to handle traffic from cgroups.
   Each rule has:
   - `match`: Required. A regex pattern to match cgroup paths.
   - One of the following actions:
     - `tproxy`: The name of the TPROXY server to use
     - `drop`: Set to true to drop the traffic
     - `direct`: Set to true to bypass the traffic

6. `route-table`:
   Required. The route table number that cgtproxy will create.
   This table will be removed when cgtproxy stops.

## Environment Variables

The following environment variables can be used to control cgtproxy:

1. `LOG_LEVEL`:
   Controls the logging verbosity.
   Values: "debug", "trace", or other standard log levels.
   Default: "info"

2. `CGTPROXY_MONITOR_BUFFER_SIZE`:
   Controls the buffer size for cgroup filesystem events.
   Value: A positive integer.
   Default: 1024
   Note: Increase this value if you experience event loss
   under heavy cgroup creation/deletion load.

## Example

Here's a minimal configuration example:

```yaml
version: 1
cgroup-root: AUTO
route-table: 300
bypass:
  - 127.0.0.1/8
tproxies:
  main:
    port: 12345
    mark: 1
rules:
  - match: /user.slice/.*
    tproxy: main
  - match: /.*
    direct: true
```

For more examples and best practices,
check the [example configuration](../misc/config/example.yaml).
