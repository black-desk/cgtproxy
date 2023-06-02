package core

import (
	"errors"
	"fmt"
	"os"
)

var (
	ErrConfigMissing = errors.New("Config is missing.")
)

type ErrCancelBySignal struct {
	os.Signal
}

func (e *ErrCancelBySignal) Error() string {
	return fmt.Sprintf("Cancelled by signal (%v).", e.Signal)
}
