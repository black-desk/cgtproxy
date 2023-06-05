//go:build debug
// +build debug

package logger

import (
	zapjournal "github.com/black-desk/zap-journal"
	"go.uber.org/zap"
	"log"
)

var Log *zap.SugaredLogger

func init() {
	var (
		logger *zap.Logger
		err    error
	)

	defer func() {
		Log = logger.Sugar()
	}()

	logger, err = zapjournal.NewDebug()
	if err == nil {
		return
	}

	log.Default().Printf("Failed to use zap-journal: %v", err)
	log.Default().Printf("Fallback to zap default development logger.")

	logger, err = zap.NewDevelopment()
	if err == nil {
		return
	}

	log.Default().Printf("Failed to use zap development logger: %v", err)
	log.Default().Printf("Disable logging.")
	logger = zap.NewNop()
}
