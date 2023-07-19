package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/black-desk/cgtproxy/internal/consts"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/core"
	"github.com/black-desk/lib/go/logger"
	"github.com/spf13/cobra"
)

var flags struct {
	CfgPath string
}

var rootCmd = &cobra.Command{
	Use:   "cgtproxy",
	Short: "A transparent network proxy manager for deepin",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err == nil {
				return
			}

			err = fmt.Errorf(
				"\n\n%w\n"+consts.CheckDocumentString,
				err,
			)

			return
		}()
		err = rootCmdRun()
		return
	},
}

func rootCmdRun() (err error) {
	log := logger.Get("cgtproxy")

	content, err := os.ReadFile(flags.CfgPath)
	if errors.Is(err, os.ErrNotExist) && flags.CfgPath == consts.CgtproxyCfgPath {
		log.Errorw("Configuration file missing fallback to default config.")

		content = []byte(config.DefaultConfig)
		err = nil
	} else if err != nil {
		log.Errorw("Failed to read configuration from file",
			"file", flags.CfgPath,
			"error", err)

		return err
	}

	var cfg *config.Config

	cfg, err = config.New(
		config.WithContent(content),
		config.WithLogger(log),
	)
	if err != nil {
		return
	}

	c, err := core.New(
		core.WithConfig(cfg),
		core.WithLogger(log),
	)
	if err != nil {
		return
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

		sig := <-sigCh
		c.Stop(&ErrCancelBySignal{Signal: sig})
	}()

	err = c.Run()
	if err == nil {
		return
	}

	log.Debugw(
		"Core exited with error.",
		"error", err,
	)

	var cancelBySignal *ErrCancelBySignal
	if errors.As(err, &cancelBySignal) {
		log.Infow("Signal received, exiting...",
			"signal", cancelBySignal.Signal,
		)
		err = nil
		return
	}

	return
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cfgPath := os.Getenv("CONFIGURATION_DIRECTORY")
	if cfgPath == "" {
		cfgPath = consts.CgtproxyCfgPath
	} else {
		cfgPath += "/config.yaml"
	}

	rootCmd.PersistentFlags().StringVarP(
		&flags.CfgPath,
		"config", "c", cfgPath,
		"the configure file to use",
	)
}
