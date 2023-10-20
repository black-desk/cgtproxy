// Code generated by interfacer; DO NOT EDIT

package interfaces

import (
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/nftman"
)

// NFTMan is an interface generated for "github.com/black-desk/cgtproxy/pkg/nftman.NFTMan".
type NFTMan interface {
	AddCgroup(string, *nftman.Target) error
	AddChainAndRulesForTProxy(*config.TProxy) error
	Clear() error
	RemoveCgroup(string) error
}
