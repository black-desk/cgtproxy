package cmd

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/black-desk/cgtproxy/internal/consts"
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

			err = fmt.Errorf("\n\n%w\n"+consts.CheckDocumentString, err)

			return
		}()

		err = checkKernelCmdRun()
		return
	},
}

func checkKernelCmdRun() (err error) {
	defer Wrap(&err, "Failed to check kernel config.")

	{ // check kernel config
		var configFile *os.File
		configFile, err = os.Open("/proc/config.gz")
		if err != nil {
			return
		}
		defer configFile.Close()

		var gzipReader io.Reader
		gzipReader, err = gzip.NewReader(configFile)
		if err != nil {
			return
		}

		scanner := bufio.NewScanner(gzipReader)
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
			if len(components) != 6 {
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
