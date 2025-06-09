// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
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

var (
	GitDescription = ""
	Version        = "dev"
)

var flags struct {
	cfgPath            string
	cpuProfile         string
	blockProfile       string
	lastingNetlinkConn bool
}

var rootCmd = &cobra.Command{
	Version: func() string {
		if GitDescription == "" {
			return Version
		}
		return Version + " ( git describe: " + GitDescription + " )"
	}(),
	Use:   "cgtproxy",
	Short: "A transparent network proxy manager.",
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

func createProfileFile(tmplStr string) (ret *os.File, err error) {
	var path string
	path, err = generateProfileName(tmplStr)
	dir := filepath.Dir(path)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return
	}

	var profile *os.File
	profile, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		return
	}

	ret = profile
	return
}

func rootCmdRun() (err error) {
	log := logger.Get("cgtproxy")

	profiles := strings.Split(os.Getenv("CGTPROXY_PROFILE"), ",")

	if slices.Contains(profiles, "cpu") {
		var cpuProfile *os.File
		cpuProfile, err = createProfileFile(flags.cpuProfile)
		if err != nil {
			return
		}
		defer cpuProfile.Close()

		err = pprof.StartCPUProfile(cpuProfile)
		defer pprof.StopCPUProfile()
		if err != nil {
			return
		}
	}

	if slices.Contains(profiles, "block") {
		var blockProfile *os.File
		blockProfile, err = createProfileFile(flags.blockProfile)
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

	content, err := os.ReadFile(flags.cfgPath)
	if errors.Is(err, os.ErrNotExist) && flags.cfgPath == CGTProxyCfgPath {
		log.Errorw("Configuration file missing fallback to default config.")

		content = []byte(config.DefaultConfig)
		err = nil
	} else if err != nil {
		log.Errorw("Failed to read configuration from file",
			"file", flags.cfgPath,
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
	if flags.lastingNetlinkConn {
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

	err = c.RunCGTProxy(ctx)
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
		&flags.cfgPath,
		"config", "c", cfgPath,
		"the configure file to use",
	)

	rootCmd.PersistentFlags().StringVar(
		&flags.cpuProfile,
		"cpu-profile", "/tmp/io.github.black-desk.cgtproxy/profiles/{{.PID}}.cpuprofile",
		""+
			"the template string use to create the cpu profile "+
			"NOTE: this option only takes effect "+
			"when $CGTPROXY_PROFILE=cpu",
	)

	rootCmd.PersistentFlags().StringVar(
		&flags.blockProfile,
		"block-profile", "/tmp/io.github.black-desk.cgtproxy/profiles/{{.PID}}.blockprofile",
		""+
			"the template string use to create the block profile "+
			"NOTE: this option only takes effect "+
			"when $CGTPROXY_PROFILE=block",
	)

	rootCmd.PersistentFlags().BoolVar(
		&flags.lastingNetlinkConn,
		"reuse-netlink-socket", true,
		"use lasting netlink socket",
	)
}
