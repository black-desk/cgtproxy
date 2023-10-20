package lastingconnector

import "github.com/google/nftables"

type LastingConnector struct {
	conn *nftables.Conn
}

type Opt = (func(*LastingConnector) (*LastingConnector, error))

func New(...Opt) (ret *LastingConnector, err error) {
	return &LastingConnector{}, nil
}
