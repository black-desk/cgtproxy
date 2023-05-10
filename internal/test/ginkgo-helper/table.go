package ginkgohelper

import (
	"fmt"
	"reflect"

	"github.com/onsi/ginkgo/v2"
)

type ContextTableEntryT struct {
	fmtArgs []any
	args    []reflect.Value
}

func ContextTable(message string, fn any, entries ...*ContextTableEntryT) {
	for i := range entries {
		ginkgo.Context(fmt.Sprintf(message, entries[i].fmtArgs...), func() {
			vfn := reflect.ValueOf(fn)
			for j := range entries[i].args {
				if entries[i].args[j].IsValid() {
					continue
				}

				entries[i].args[j] = reflect.New(vfn.Type().In(j)).Elem()
			}
			vfn.Call(entries[i].args)
		})
	}
}

func (c *ContextTableEntryT) WithFmt(args ...any) *ContextTableEntryT {
	c.fmtArgs = args
	return c
}

func ContextTableEntry(args ...any) *ContextTableEntryT {
	ret := &ContextTableEntryT{}
	for i := range args {
		ret.args = append(ret.args, reflect.ValueOf(args[i]))
	}
	ret.fmtArgs = args
	return ret
}
