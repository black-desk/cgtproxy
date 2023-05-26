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

	logger, err = zapjournal.NewDebug()
	if err != nil {
		log.Default().Printf("Failed to use zap-journal:\n%s", err.Error())
		log.Default().Printf("Fallback to zap default production logger")

		logger, err = zap.NewDevelopment()
		if err != nil {
			panic("Failed to use zap production logger:\n" + err.Error())
		}
	}

	Log = logger.Sugar()
}
