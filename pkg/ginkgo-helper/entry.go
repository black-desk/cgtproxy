package ginkgohelper

import "reflect"

type ContextTableEntryT struct {
	fmtArgs []any
	args    []reflect.Value
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
