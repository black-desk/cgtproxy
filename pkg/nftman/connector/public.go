package connector

import (
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/google/nftables"
)

func (c *Connector) Connect() (ret *nftables.Conn, err error) {
	defer Wrap(&err, "new netlink connection")
	return nftables.New()
}

func (c *Connector) Release() error {
	return nil
}
