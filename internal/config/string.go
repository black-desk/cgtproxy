package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func (r *Rule) String() string {
	if r.Drop {
		return fmt.Sprintf("rule [ match: %s | DROP ]", r.Match)
	} else if r.Direct {
		return fmt.Sprintf("rule [ match: %s | DIRECT ]", r.Match)
	} else if r.Proxy != nil {
		return fmt.Sprintf("rule [ match: %s | PROXY %s ]",
			r.Match, r.Proxy.String())
	} else if r.TProxy != nil {
		return fmt.Sprintf("rule [ match: %s | TPROXY %s ]",
			r.Match, r.TProxy.String())
	}

	panic("this should never happened")
}

func (p *Proxy) String() string {
	var (
		bytes []byte
		err   error
	)

	if bytes, err = yaml.Marshal(p); err != nil {
		panic("this should never happened")
	}

	return fmt.Sprintf("%stproxy:\n  %s",
		string(bytes),
		strings.ReplaceAll(p.TProxy.String(), "\n", "\n  "),
	)
}

func (t *TProxy) String() string {
	var (
		bytes []byte
		err   error
	)

	if bytes, err = yaml.Marshal(t); err != nil {
		panic("this should never happened")
	}

	return fmt.Sprint(string(bytes))
}
