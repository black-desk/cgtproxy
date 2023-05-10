package ginkgohelper

import (
	"fmt"
	"reflect"

	"github.com/onsi/ginkgo/v2"
)

func ContextTable(message string, fn any, entries ...*ContextTableEntryT) {
	for i := range entries {
		ginkgo.Context(
			fmt.Sprintf(message, entries[i].fmtArgs...),
			func() { runContextTableEntry(fn, entries[i]) },
		)
	}
}

func runContextTableEntry(fn any, entry *ContextTableEntryT) {
	vfn := reflect.ValueOf(fn)
	for j := range entry.args {
		if entry.args[j].IsValid() {
			continue
		}

		entry.args[j] = reflect.New(vfn.Type().In(j)).Elem()
	}
	vfn.Call(entry.args)
}
