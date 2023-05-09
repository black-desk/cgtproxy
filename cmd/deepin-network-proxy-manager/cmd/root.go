package cmd

import (
	"fmt"
	"os"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
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

			err = fmt.Errorf("\n\n%w\n"+consts.CheckDocumentString, err)

			return
		}()
		err = rootCmdRun()
		return
	},
}

func rootCmdRun() (err error) {
	content, err := os.ReadFile(flags.CfgPath)
	if err != nil {
		log.Err().Printf(
			"Failed to read configuration from %s:\n%s",
			flags.CfgPath, err.Error(),
		)
		return err
	}

	var cfg *config.Config

	if cfg, err = config.Load(content); err != nil {
		log.Err().Printf(
			"Failed to load configuration:\n%s",
			err.Error(),
		)
		return err
	}

	core, err := core.New(
		core.WithConfig(cfg),
	)
	if err != nil {
		log.Err().Printf("Failed to init core:\n%s", err.Error())
		return err
	}

	return core.Run()
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
