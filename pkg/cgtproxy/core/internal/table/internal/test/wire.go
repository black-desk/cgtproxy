//go:build wireinject
// +build wireinject

package tabletest

import (
	"github.com/google/nftables"
	"github.com/google/wire"
)

func InjectedConn() (*nftables.Conn, error) {
	panic(wire.Build(set))
}
