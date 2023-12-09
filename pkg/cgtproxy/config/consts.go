package config

const (
	DefaultConfig = `
version: 1
cgroup-root: AUTO
route-table: 400
rules:
  - match: \/.*
    direct: true
`
	IPv4LocalhostStr = "127.0.0.1"
)
