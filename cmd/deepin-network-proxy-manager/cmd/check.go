package cmd

import (
	"fmt"

	"github.com/black-desk/deepin-network-proxy-manager/internal/consts"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check system requirements",
	Long:  `Check kernel configuration, cgorup v2 mount and configuration.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err == nil {
				return
			}

			err = fmt.Errorf("\n\n%w\n"+consts.CheckDocumentString, err)

			return
		}()

		err = checkCmdRun()
		return
	},
}

func checkCmdRun() (err error) {
	if err = checkKernelCmdRun(); err != nil {
		return
	}

	if err = checkConfigCmdRun(); err != nil {
		return
	}

	return
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
