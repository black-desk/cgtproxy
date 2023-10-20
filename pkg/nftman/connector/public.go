package connector

import "github.com/google/nftables"

func (c *Connector) Connect() (*nftables.Conn, error) {
	return nftables.New()
}

func (c *Connector) Release() error {
	return nil
}
