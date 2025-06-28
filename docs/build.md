<!--
SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

# Build Guide

en | [zh_CN](./build.zh_CN.md)

<!-- Do not remove this warning when updating documentation -->

> [!WARNING]
>
> This English documentation is translated from the Chinese version using AI and
> may contain errors.

The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**,
**SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this
document are to be interpreted as described in [RFC 2119][rfc-2119].

[rfc-2119]: https://datatracker.ietf.org/doc/html/rfc2119

---

You **SHOULD** use `make` to build cgtproxy.

It is **NOT RECOMMENDED** to use `go build ./cmd/cgtproxy` to compile this
project.

## Testing

To avoid breaking nft configuration, a large portion of tests should run in a
network namespace. `make test` will create a network namespace and set an
environment variable to enable these tests. You can check the
[Makefile](../Makefile) and [test source code](../pkg/nftman/nftman_test.go) for
details.
