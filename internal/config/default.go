package config

const DefaultConfig = `
version: 1
cgroup-root: AUTO
marks: '[3000,3100)'
rules:
  - match: \/.*
    direct: true
`
