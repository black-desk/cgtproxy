# Build guide

It is ok to use `go build ./cmd/cgtproxy` to build this project
if you want to install it from source.

But it's recommend to use the [Makefile] I provided to make sure
execute `go generate` before buid.

[Makefile]: ../Makefile

## test

Some tests need to run in network namespace
to void mess up your nft configuration.

These tests will be skipped when running without a environment variable set.

Check [sources] for details.

[sources]: ../pkg/nftman/nftman_test.go

To run these tests locally,
use the `test` target defined in [Makefile] by running `make test`.
