package cmd

import (
	"fmt"
	"os"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/black-desk/lib/go/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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

			err = fmt.Errorf("\n%w\n"+CheckDocumentString, err)

			return
		}()

		err = checkConfigCmdRun()
		return
	},
}

func checkConfigCmdRun() (err error) {
	defer Wrap(&err)

	log := zap.NewNop().Sugar()
	if checkFlags.EnableLogger {
		log = logger.Get("cgtproxy")
	}

	var content []byte
	content, err = os.ReadFile(flags.CfgPath)
	if err != nil {
		Wrap(
			&err,
			"read configuration from %s",
			flags.CfgPath,
		)
		return
	}

	_, err = config.New(
		config.WithContent(content),
		config.WithLogger(log),
	)
	if err != nil {
		return
	}

	return
}

func init() {
	checkCmd.AddCommand(checkConfigCmd)
}
