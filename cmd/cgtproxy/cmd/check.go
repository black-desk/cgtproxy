package cmd

import (
	"fmt"

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

			err = fmt.Errorf("\n\n%w\n"+CheckDocumentString, err)

			return
		}()

		err = checkCmdRun()
		return
	},
}

func checkCmdRun() (err error) {
	err = checkKernelCmdRun()
	if err != nil {
		return
	}

	err = checkConfigCmdRun()
	if err != nil {
		return
	}

	return
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
