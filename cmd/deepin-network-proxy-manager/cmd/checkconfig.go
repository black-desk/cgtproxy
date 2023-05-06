package cmd

import (
	"fmt"
	"os"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// checkConfigCmd represents the config command
var checkConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Check configuration",
	Long:  `Validate configuration.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = checkConfigCmdRun()
		return
	},
}

func checkConfigCmdRun() (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(
			"failed to validate configuration: %w",
			err,
		)
	}()

	var content []byte
	if content, err = os.ReadFile(flags.CfgPath); err != nil {
		err = fmt.Errorf(
			"failed to read configuration [ %s ]: %w",
			flags.CfgPath, err,
		)
		return
	}

	var cfg *config.Config
	if err = yaml.Unmarshal(content, &cfg); err != nil {
		err = fmt.Errorf("failed to unmarshal configuration: %w", err)
		return
	}

	if err = cfg.Check(); err != nil {
		return
	}

	return
}

func init() {
	checkCmd.AddCommand(checkConfigCmd)
}
