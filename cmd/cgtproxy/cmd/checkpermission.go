package cmd

import (
	"errors"
	"fmt"

	. "github.com/black-desk/lib/go/errwrap"
	"github.com/spf13/cobra"
	"kernel.org/pub/linux/libs/security/libcap/cap"
)

// checkPermissionCmd represents the permission command
var checkPermissionCmd = &cobra.Command{
	Use:   "permission",
	Short: "Check permission",
	Long:  `Check cgtproxy have get all required capabilities.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err == nil {
				return
			}

			err = fmt.Errorf("\n\n%w\n"+CheckDocumentString, err)

			return
		}()

		err = checkPermissionCmdRun()
		return
	},
}

func checkPermissionCmdRun() (err error) {
	defer Wrap(&err)
	capSet := cap.GetProc()
	hasCapNetAdmin := false
	hasCapNetAdmin, err = capSet.GetFlag(cap.Effective, cap.NET_ADMIN)
	if err != nil {
		return
	}

	if !hasCapNetAdmin {
		err = errors.New("CAP_NET_ADMIN is required to update nftable.")
		return
	}

	return
}

func init() {
	checkCmd.AddCommand(checkPermissionCmd)
}
