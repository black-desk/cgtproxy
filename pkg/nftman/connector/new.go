package connector

type Connector struct{}

type Opt = (func(*Connector) (*Connector, error))

//go:generate go run github.com/rjeczalik/interfaces/cmd/interfacer@v0.3.0 -for github.com/black-desk/cgtproxy/pkg/nftman/connector.Connector -as interfaces.NetlinkConnector -o ../../interfaces/netlinkconnector.go

func New(...Opt) (ret *Connector, err error) {
	return &Connector{}, nil
}
