package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/spf13/cobra"
)

var flags struct {
	CfgPath string
}

var rootCmd = &cobra.Command{
	Use:   "deepin-network-proxy-manager",
	Short: "A transparent network proxy manager for deepin",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err == nil {
				return
			}

			Log.Errorw("Error occur while running deepin-network-proxy-manager.",
				"config", flags.CfgPath,
				"error", err,
			)

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
	content, err := os.ReadFile(flags.CfgPath)
	if err != nil {
		Log.Errorw("Failed to read configuration from file",
			"file", flags.CfgPath,
			"error", err)
		return err
	}

	var cfg *config.Config

	cfg, err = config.Load(content)
	if err != nil {
		return
	}

	c, err := core.New(
		core.WithConfig(cfg),
	)
	if err != nil {
		return
	}

	err = c.Run()
	if err == nil {
		return
	}

	Log.Debugw(
		"Core exited with error.",
		"error", err,
	)

	{
		var cancelBySignal *core.ErrCancelBySignal
		if errors.As(err, &cancelBySignal) {
			Log.Infow("Signal received, exiting...",
				"signal", cancelBySignal.Signal,
			)
			err = nil
			return
		}
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
	rootCmd.PersistentFlags().StringVarP(
		&flags.CfgPath,
		"config", "c", consts.CfgPath,
		"the configure file to use",
	)
}
