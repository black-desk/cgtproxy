// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var checkFlags struct {
	EnableLogger bool
}

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

			err = fmt.Errorf("\n%w\n"+CheckDocumentString, err)

			return
		}()

		err = checkCmdRun()
		return
	},
}

func checkCmdRun() (err error) {
	err = checkConfigCmdRun()
	if err != nil {
		return
	}

	err = checkPermissionCmdRun()
	if err != nil {
		return
	}

	return
}

func init() {
	checkCmd.PersistentFlags().BoolVar(
		&checkFlags.EnableLogger,
		"with-logger", false,
		"enable logger during check configuration",
	)

	rootCmd.AddCommand(checkCmd)
}
