package location

import (
	"fmt"
	"path"
	"runtime"
)

func Capture() string {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		panic("this should never happened.")
	}

	return fmt.Sprintf("%s:%d(%x)\n\t", path.Base(file), line, pc)
}
