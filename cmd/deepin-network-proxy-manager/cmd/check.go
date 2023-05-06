package cmd

import (
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check system requirements",
	Long:  `Check kernel configuration, cgorup v2 mount and configuration.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
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
