package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"text/template"

	"github.com/black-desk/cgtproxy/internal/consts"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy"
	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/lib/go/logger"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var flags struct {
	CfgPath    string
	CPUProfile string
}

var rootCmd = &cobra.Command{
	Use:   "cgtproxy",
	Short: "A transparent network proxy manager for deepin",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err == nil {
				return
			}

			err = fmt.Errorf(
				"\n\n%w\n"+consts.CheckDocumentString,
				err,
			)

			return
		}()
		err = rootCmdRun()
		return
	},
}

func generateCPUProfileName(tmplStr string) (name string, err error) {
	var tmpl *template.Template
	tmpl = template.New("cpu profile name")
	tmpl, err = tmpl.Parse(flags.CPUProfile)
	if err != nil {
		return
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, map[string]int{"PID": os.Getpid()})
	if err != nil {
		return
	}

	name = buf.String()
	return
}

func rootCmdRun() (err error) {
	log := logger.Get("cgtproxy")

	if slices.Contains(
		strings.Split(os.Getenv("CGTPROXY_PROFILE"), ","),
		"cpu",
	) {
		var path string
		path, err = generateCPUProfileName(flags.CPUProfile)
		if err != nil {
			return
		}

		var cpuProfile *os.File
		cpuProfile, err = os.Create(path)
		defer cpuProfile.Close()
		if err != nil {
			return
		}

		err = pprof.StartCPUProfile(cpuProfile)
		defer pprof.StopCPUProfile()
		if err != nil {
			return
		}
	}

	content, err := os.ReadFile(flags.CfgPath)
	if errors.Is(err, os.ErrNotExist) && flags.CfgPath == consts.CgtproxyCfgPath {
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

	c, err := cgtproxy.New(
		cgtproxy.WithConfig(cfg),
		cgtproxy.WithLogger(log),
	)
	if err != nil {
		return
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

		sig := <-sigCh
		c.Stop(&ErrCancelBySignal{Signal: sig})
	}()

	err = c.Run()
	if err == nil {
		return
	}

	log.Debugw(
		"Core exited with error.",
		"error", err,
	)

	var cancelBySignal *ErrCancelBySignal
	if errors.As(err, &cancelBySignal) {
		log.Infow("Signal received, exiting...",
			"signal", cancelBySignal.Signal,
		)
		err = nil
		return
	}

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
		cfgPath = consts.CgtproxyCfgPath
	} else {
		cfgPath += "/config.yaml"
	}

	rootCmd.PersistentFlags().StringVarP(
		&flags.CfgPath,
		"config", "c", cfgPath,
		"the configure file to use",
	)

	rootCmd.PersistentFlags().StringVarP(
		&flags.CPUProfile,
		"cpu-profile", "", "/tmp/cgtproxy.{{.PID}}.cpuprofile",
		""+
			"the template string use to create the cpu profile "+
			"NOTE: this option only takes effect "+
			"when $CGTPROXY_PROFILE=cpu",
	)
}
