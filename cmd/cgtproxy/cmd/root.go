package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"text/template"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/interfaces"
	"github.com/black-desk/lib/go/logger"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

var flags struct {
	CfgPath            string
	CPUProfile         string
	BlockProfile       string
	LastingNetlinkConn bool
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
				"\n%w\n\n"+
					`You can run 'cgtproxy check' to perform health check built in cgtproxy.`+
					CheckDocumentString,
				err,
			)

			return
		}()
		err = rootCmdRun()
		return
	},
}

func generateProfileName(tmplStr string) (name string, err error) {
	var tmpl *template.Template
	tmpl = template.New("profile name")
	tmpl, err = tmpl.Parse(tmplStr)
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

	profiles := strings.Split(os.Getenv("CGTPROXY_PROFILE"), ",")

	if slices.Contains(profiles, "cpu") {
		var path string
		path, err = generateProfileName(flags.CPUProfile)
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

	if slices.Contains(profiles, "block") {
		var path string
		path, err = generateProfileName(flags.BlockProfile)
		if err != nil {
			return
		}

		var blockProfile *os.File
		blockProfile, err = os.Create(path)
		defer blockProfile.Close()
		if err != nil {
			return
		}

		runtime.SetBlockProfileRate(1)
		defer pprof.Lookup("block").WriteTo(blockProfile, 0)
		if err != nil {
			return
		}
	}

	content, err := os.ReadFile(flags.CfgPath)
	if errors.Is(err, os.ErrNotExist) && flags.CfgPath == CGTProxyCfgPath {
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

	var c interfaces.CGTProxy
	if flags.LastingNetlinkConn {
		c, err = injectedLastingCGTProxy(cfg, log)
	} else {
		c, err = injectedCGTProxy(cfg, log)
	}

	if err != nil {
		return
	}

	ctx, cancel := context.WithCancelCause(context.Background())

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

		sig := <-sigCh
		cancel(&ErrCancelBySignal{sig})
	}()

	err = c.Run(ctx)
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
		cfgPath = CGTProxyCfgPath
	} else {
		cfgPath += "/config.yaml"
	}

	rootCmd.PersistentFlags().StringVarP(
		&flags.CfgPath,
		"config", "c", cfgPath,
		"the configure file to use",
	)

	rootCmd.PersistentFlags().StringVar(
		&flags.CPUProfile,
		"cpu-profile", "/tmp/cgtproxy.{{.PID}}.cpuprofile",
		""+
			"the template string use to create the cpu profile "+
			"NOTE: this option only takes effect "+
			"when $CGTPROXY_PROFILE=cpu",
	)

	rootCmd.PersistentFlags().StringVar(
		&flags.BlockProfile,
		"block-profile", "/tmp/cgtproxy.{{.PID}}.blockprofile",
		""+
			"the template string use to create the block profile "+
			"NOTE: this option only takes effect "+
			"when $CGTPROXY_PROFILE=block",
	)

	rootCmd.PersistentFlags().BoolVar(
		&flags.LastingNetlinkConn,
		"reuse-netlink-socket", false,
		"the configure file to use",
	)
}
