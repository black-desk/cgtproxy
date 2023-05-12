package cmd

import (
	"fmt"
	"os"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	"github.com/black-desk/deepin-network-proxy-manager/pkg/location"
	"github.com/spf13/cobra"
)

// checkConfigCmd represents the config command
var checkConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Check configuration",
	Long:  `Validate configuration.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err == nil {
				return
			}

			err = fmt.Errorf("\n\n%w\n"+consts.CheckDocumentString, err)

			return
		}()

		err = checkConfigCmdRun()
		return
	},
}

func checkConfigCmdRun() (err error) {
	defer func() {
		if err == nil {
			return
		}
		err = fmt.Errorf(location.Catch()+
			"failed to validate configuration:\n%w",
			err,
		)
	}()

	var content []byte
	content, err = os.ReadFile(flags.CfgPath)
	if err != nil {
		err = fmt.Errorf(location.Catch()+
			"failed to read configuration [ %s ]:\n%w",
			flags.CfgPath, err,
		)
		return
	}

	_, err = config.Load(content)
	if err != nil {
		return
	}

	return
}

func init() {
	checkCmd.AddCommand(checkConfigCmd)
}
