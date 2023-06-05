//go:build !debug
// +build !debug

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

	logger, err = zapjournal.New()
	if err == nil {
		return
	}

	log.Default().Printf("Failed to use zap-journal: %v", err)
	log.Default().Printf("Fallback to zap default production logger.")

	logger, err = zap.NewProduction()
	if err == nil {
		return
	}

	log.Default().Printf("Failed to use zap production logger: %v", err)
	log.Default().Printf("Disable logging.")
	logger = zap.NewNop()
}
