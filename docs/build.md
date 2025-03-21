# Build Guide

It is ok to use `go build ./cmd/cgtproxy` to build this project
if you want to install it from source.

But it's recommended to use the [Makefile]
to make sure `go generate` is executed before build.

[Makefile]: ../Makefile

## Testing

Some tests need to run in a network namespace
to avoid messing up your nft configuration.

These tests will be skipped
when running without the required environment variable set.

Check [sources] for details.

[sources]: ../pkg/nftman/nftman_test.go

To run these tests locally,
use the `test` target defined in [Makefile]
by running `make test`.
