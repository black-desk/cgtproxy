package main

import (
	zapjournal "github.com/black-desk/deepin-network-proxy-manager/pkg/zap-journal"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/zap-journal/conn"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/zap-journal/encoder"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	{
		defaultJournalLogger, err := zapjournal.New()
		if err != nil {
			panic(err)
		}

		defaultJournalLogger.Info("some log message")
	}

	{
		enc, err := encoder.New(encoder.WithCfg(zapcore.EncoderConfig{
			StacktraceKey: "STACK_TRACE",
		}))
		if err != nil {
			panic(err)
		}

		sink, err := conn.New(conn.WithAddress("/run/systemd/journal/socket"))
		if err != nil {
			panic(err)
		}

		level := zap.NewAtomicLevel()

		journalLogger := zap.New(zapcore.NewCore(enc, sink, level))

		journalLogger = journalLogger.Named("your syslog identifier")

		journalLogger.Info("some log message")
	}
}
