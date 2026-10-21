// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

// cgroupKeys reads the expected inode keys independently from the application.
// The root uses a fallback chain rather than an entry in cgroup-vmap.
func cgroupKeys(root string) (map[string]string, error) {
	keys := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == root {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		key := binaryutil.NativeEndian.PutUint64(info.Sys().(*syscall.Stat_t).Ino)
		keys[string(key)] = path
		return nil
	})
	return keys, err
}

// Check every key and verdict, not just the number of elements: Go map order
// makes the particular entries lost to a truncated message unpredictable.
func checkCgroupMap(args []string) error {
	flags := flag.NewFlagSet("check-cgroup-map", flag.ExitOnError)
	root := flags.String("root", "/sys/fs/cgroup", "cgroup root; all descendants must have entries")
	proxyPrefix := flags.String("proxy-prefix", "group-proxy-", "basename prefix of proxy cgroups")
	proxyChain := flags.String("proxy-chain", "proxy-MARK", "expected proxy target chain")
	flags.Parse(args)
	keys, err := cgroupKeys(*root)
	if err != nil {
		return err
	}
	conn := &nftables.Conn{}
	set, err := conn.GetSetByName(&nftables.Table{Name: "cgtproxy", Family: nftables.TableFamilyINet}, "cgroup-vmap")
	if err != nil {
		return err
	}
	elements, err := conn.GetSetElements(set)
	if err != nil {
		return err
	}
	if len(elements) != len(keys) {
		return fmt.Errorf("cgroup map has %d elements; want %d", len(elements), len(keys))
	}
	for _, element := range elements {
		path, ok := keys[string(element.Key)]
		if !ok {
			return fmt.Errorf("unexpected or duplicate cgroup key %x", element.Key)
		}
		want := expr.Verdict{Kind: expr.VerdictReturn}
		switch name := filepath.Base(path); {
		case strings.HasPrefix(name, *proxyPrefix):
			want.Kind, want.Chain = expr.VerdictGoto, *proxyChain
		}
		got, err := decodeSetVerdict(element.Val)
		if err != nil {
			return fmt.Errorf("cgroup %s: %w", path, err)
		}
		if got != want {
			return fmt.Errorf("cgroup %s has verdict %+v; want %+v", path, got, want)
		}
		delete(keys, string(element.Key))
	}
	return nil
}

// nftables v0.3.0 returns a set verdict's nested attributes in Val without
// decoding VerdictData. Decode the kernel response, including signed codes.
func decodeSetVerdict(data []byte) (expr.Verdict, error) {
	var verdict expr.Verdict
	decoder, err := netlink.NewAttributeDecoder(data)
	if err != nil {
		return verdict, err
	}
	decoder.ByteOrder = binary.BigEndian
	foundCode := false
	for decoder.Next() {
		switch decoder.Type() {
		case unix.NFTA_VERDICT_CODE:
			verdict.Kind = expr.VerdictKind(int32(decoder.Uint32()))
			foundCode = true
		case unix.NFTA_VERDICT_CHAIN:
			verdict.Chain = decoder.String()
		}
	}
	if err := decoder.Err(); err != nil {
		return verdict, err
	}
	if !foundCode {
		return verdict, fmt.Errorf("set element has no verdict code")
	}
	return verdict, nil
}
