package logger

import (
	"github.com/black-desk/lib/go/logger"
	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func init() {
	Log = logger.Get("cgtproxy")
}
