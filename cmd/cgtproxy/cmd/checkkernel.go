package cmd

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	. "github.com/black-desk/lib/go/errwrap"
	"github.com/spf13/cobra"
)

// checkKernelCmd represents the kernel command
var checkKernelCmd = &cobra.Command{
	Use:   "kernel",
	Short: "Check kernel configuration",
	Long:  `Check required kernel features.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		defer func() {
			if err == nil {
				return
			}

			err = fmt.Errorf("\n%w\n"+CheckDocumentString, err)

			return
		}()

		err = checkKernelCmdRun()
		return
	},
}

func getKernelConfigFromProcFS() (ret io.ReadCloser, err error) {
	defer Wrap(&err, "get kernel config from /proc/config.gz")

	var configDotGz *os.File
	configDotGz, err = os.Open("/proc/config.gz")
	if err != nil {
		return
	}

	var config io.ReadCloser
	config, err = gzip.NewReader(configDotGz)
	if err != nil {
		return
	}

	ret = config
	return
}

func sliceToStr[T int8 | uint8](ints []T) string {
	b := make([]byte, 0, len(ints))
	for _, v := range ints {
		if v == 0 {
			break
		}
		b = append(b, byte(v))
	}
	return string(b)
}

func getKernelConfigFromBoot() (ret io.ReadCloser, err error) {
	defer Wrap(&err, "get kernel config from /boot/config-*")

	var uname syscall.Utsname
	err = syscall.Uname(&uname)
	if err != nil {
		Wrap(&err, "uname")
		return
	}

	var config io.ReadCloser
	config, err = os.Open(fmt.Sprintf(
		"/boot/config-%s",
		sliceToStr(uname.Release[:]),
	))
	if err != nil {
		return
	}

	ret = config
	return
}

func getKernelConfig() (ret io.ReadCloser, err error) {
	var config io.ReadCloser
	config, err = getKernelConfigFromProcFS()

	if errors.Is(err, os.ErrNotExist) {
		procfsError := err.Error()
		config, err = getKernelConfigFromBoot()
		if err != nil {
			Wrap(&err, procfsError+" fallback")
		}
	}
	if err != nil {
		return
	}

	ret = config
	return

}

func checkKernelConfig() (err error) {
	defer Wrap(&err, "check kernel config")

	var config io.ReadCloser
	config, err = getKernelConfig()
	if err != nil {
		return
	}
	defer config.Close()

	scanner := bufio.NewScanner(config)
	scanner.Split(bufio.ScanLines)

	var (
		module                        bool
		configNftTproxy               bool
		configNetfilterXtTargetTproxy bool
		configNfTproxyIpv4            bool
		configNfTproxyIpv6            bool
	)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		components := strings.SplitN(line, "=", 2)
		if len(components) != 2 {
			err = fmt.Errorf(
				"Unexpected format of /proc/config.gz (line: %s).",
				line,
			)
			Wrap(&err)
			return
		}

		if !(components[1] == "y" || components[1] == "m") {
			continue
		}

		switch components[0] {
		case "CONFIG_NFT_TPROXY":
			configNftTproxy = true
		case "CONFIG_NETFILTER_XT_TARGET_TPROXY":
			configNetfilterXtTargetTproxy = true
		case "CONFIG_NF_TPROXY_IPV4":
			configNfTproxyIpv4 = true
		case "CONFIG_NF_TPROXY_IPV6":
			configNfTproxyIpv6 = true
		default:
			continue
		}

		if components[1] == "m" {
			module = true
		}
	}

	if !configNftTproxy {
		err = errors.New("CONFIG_NFT_TPROXY is missing in kernel config.")
		return
	}
	if !configNetfilterXtTargetTproxy {
		err = errors.New("CONFIG_NETFILTER_XT_TARGET_TPROXY is missing in kernel config.")
		return
	}
	if !configNfTproxyIpv4 {
		err = errors.New("CONFIG_NF_TPROXY_IPV4 is missing in kernel config.")
		return
	}
	if !configNfTproxyIpv6 {
		err = errors.New("CONFIG_NF_TPROXY_IPV6 is missing in kernel config.")
		return
	}

	if !module {
		return
	}

	return
}

func checkKernelCmdRun() (err error) {
	defer Wrap(&err)

	err = checkKernelConfig()
	if err != nil {
		return
	}

	{ // check kernel module loaded
		var modulesFile *os.File
		modulesFile, err = os.Open("/proc/modules")
		if err != nil {
			return
		}
		defer modulesFile.Close()

		scanner := bufio.NewScanner(modulesFile)
		scanner.Split(bufio.ScanLines)

		var (
			nfTables bool
		)

		for scanner.Scan() {
			line := scanner.Text()

			components := strings.Split(line, " ")

			// https://stackoverflow.com/questions/39435927/what-is-oe-in-linux
			if len(components) < 6 {
				err = fmt.Errorf("Unexpected format of /proc/modules. (line: %s)", line)
				Wrap(&err)
				return
			}

			if components[4] != "Live" {
				continue
			}

			switch components[0] {
			case "nf_tables":
				nfTables = true
			default:
				continue
			}
		}

		if !nfTables {
			err = errors.New("kernel module `nf_tables` not loaded.")
			return
		}
	}

	return
}

func init() {
	checkCmd.AddCommand(checkKernelCmd)
}
