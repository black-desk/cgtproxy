package tabletest

import (
	"sync"
	"github.com/google/nftables"
	"github.com/google/wire"
)

var (
	_nftConnOnce sync.Once
	_nftConn     *nftables.Conn
	_nftErr      error
)

func provideNftConn() (ret *nftables.Conn, err error) {
	_nftConnOnce.Do(func() {
		_nftConn, _nftErr = nftables.New(nftables.AsLasting())
	})

	if _nftErr != nil {
		err = _nftErr
		return
	}

	ret = _nftConn
	return
}

var set = wire.NewSet(
	provideNftConn,
)
