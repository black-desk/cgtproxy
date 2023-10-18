package cmd

import (
	"fmt"
	"os"
)

type ErrCancelBySignal struct {
	os.Signal
}

func (e *ErrCancelBySignal) Error() string {
	return fmt.Sprintf("Cancelled by signal (%v).", e.Signal)
}
