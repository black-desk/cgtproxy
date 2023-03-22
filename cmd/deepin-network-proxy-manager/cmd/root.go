package cmd

import (
	"os"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var flags struct {
	CfgPath string
}

var rootCmd = &cobra.Command{
	Use:   "deepin-network-proxy-manager",
	Short: "A transparent network proxy manager for deepin",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		content, err := os.ReadFile(flags.CfgPath)
		if err != nil {
			log.Err().Printf("Failed to read configuration [ %s ]: %v", flags.CfgPath, err)
			return err
		}

		var cfg *config.Config

		err = yaml.Unmarshal(content, &cfg)
		if err != nil {
			log.Err().Printf("Failed to unmarshal configuration: %v", err)
			return err
		}

		core, err := core.New(core.WithConfig(cfg))
		if err != nil {
			log.Err().Printf("Failed to init core: %v", err)
			return err
		}

		return core.Run()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&flags.CfgPath, "config", "c", consts.CfgPath, "the configure file to use")
}
