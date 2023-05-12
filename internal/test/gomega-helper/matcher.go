package gomegahelper

import (
	"github.com/black-desk/deepin-network-proxy-manager/internal/test/gomega-helper/matchers"
	_ "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

func MatchErr(expected interface{}) types.GomegaMatcher {
	return &matchers.MatchErrMatcher{
		Expected: expected,
	}
}
