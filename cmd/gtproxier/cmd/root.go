package cmd

import (
	"errors"
	"os"

	"github.com/black-desk/cgtproxy/internal/consts"
	"github.com/black-desk/cgtproxy/pkg/gtproxier"
	"github.com/black-desk/cgtproxy/pkg/gtproxier/config"
	"github.com/black-desk/lib/go/logger"
	"github.com/spf13/cobra"
)

var flags struct {
	CfgPath string
}

var rootCmd = &cobra.Command{
	Use:   "gtproxier",
	Short: "A simple golang program forwarding TPROXY traffics to http/socks proxy server.",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = rootCmdRun()
		return
	},
}

func rootCmdRun() (err error) {
	log := logger.Get("gtproxier")

	var content []byte
	content, err = os.ReadFile(flags.CfgPath)
	if errors.Is(err, os.ErrNotExist) && flags.CfgPath == consts.CGTProxyCfgPath {
		log.Errorw("Configuration file missing fallback to default config.")

		content = []byte(config.DefaultConfig)
		err = nil
	} else if err != nil {
		log.Errorw("Failed to read configuration from file",
			"file", flags.CfgPath,
			"error", err)

		return err
	}

	var cfg *config.Config
	cfg, err = config.New(
		config.WithContent(content),
		config.WithLogger(log),
	)
	if err != nil {
		return
	}

	_, err = gtproxier.New(gtproxier.WithConfig(cfg), gtproxier.WithLogger(log))

	return

}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cfgPath := os.Getenv("CONFIGURATION_DIRECTORY")
	if cfgPath == "" {
		cfgPath = consts.CGTProxyCfgPath
	} else {
		cfgPath += "/config.yaml"
	}

	rootCmd.PersistentFlags().StringVarP(
		&flags.CfgPath,
		"config", "c", cfgPath,
		"the configure file to use",
	)
}
