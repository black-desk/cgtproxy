// SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

// fixture supplies ordinary clients and transparent echo servers for shell cases.
// It deliberately does not import cgtproxy's implementation packages.
package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type receipt struct {
	Server      string `json:"server"`
	Destination string `json:"destination"`
	Payload     string `json:"payload"`
}

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("usage: fixture serve|client|check-nft-init [flags]"))
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = serve(os.Args[2:])
	case "client":
		err = client(os.Args[2:])
	case "check-nft-init":
		err = checkNFTInit(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func socketControl(transparent bool) func(string, string, syscall.RawConn) error {
	return func(network, _ string, raw syscall.RawConn) error {
		var socketErr error
		err := raw.Control(func(fd uintptr) {
			level, transparentOption, originalOption := unix.SOL_IP, unix.IP_TRANSPARENT, unix.IP_RECVORIGDSTADDR
			if strings.HasSuffix(network, "6") {
				level, transparentOption, originalOption = unix.SOL_IPV6, unix.IPV6_TRANSPARENT, unix.IPV6_RECVORIGDSTADDR
			}
			options := [][2]int{{unix.SOL_SOCKET, unix.SO_REUSEADDR}}
			if transparent {
				options = append(options, [2]int{level, transparentOption})
			}
			if strings.HasPrefix(network, "udp") {
				options = append(options, [2]int{level, originalOption})
			}
			for _, option := range options {
				if socketErr = unix.SetsockoptInt(int(fd), option[0], option[1], 1); socketErr != nil {
					return
				}
			}
		})
		return errors.Join(err, socketErr)
	}
}

func serve(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	network := flags.String("network", "tcp4", "tcp4, tcp6, udp4 or udp6")
	address := flags.String("address", "0.0.0.0:17893", "listen address")
	id := flags.String("id", "proxy", "server identity returned to clients")
	transparent := flags.Bool("transparent", false, "enable transparent sockets")
	ready := flags.String("ready", "", "file created after listening")
	flags.Parse(args)
	lc := net.ListenConfig{Control: socketControl(*transparent)}
	var mu sync.Mutex
	record := func(value receipt) {
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewEncoder(os.Stdout).Encode(value)
	}
	markReady := func() error { return os.WriteFile(*ready, nil, 0600) }
	if strings.HasPrefix(*network, "tcp") {
		listener, err := lc.Listen(context.Background(), *network, *address)
		if err != nil {
			return err
		}
		defer listener.Close()
		if err := markReady(); err != nil {
			return err
		}
		for {
			conn, err := listener.Accept()
			if err != nil {
				return err
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				scanner := bufio.NewScanner(conn)
				for scanner.Scan() {
					value := receipt{*id, conn.LocalAddr().String(), scanner.Text()}
					record(value)
					if err := json.NewEncoder(conn).Encode(value); err != nil {
						return
					}
				}
			}()
		}
	}
	packet, err := lc.ListenPacket(context.Background(), *network, *address)
	if err != nil {
		return err
	}
	defer packet.Close()
	if err := markReady(); err != nil {
		return err
	}
	conn := packet.(*net.UDPConn)
	for {
		buf, oob := make([]byte, 4096), make([]byte, 256)
		n, oobn, _, peer, err := conn.ReadMsgUDP(buf, oob)
		if err != nil {
			return err
		}
		destination, err := originalDestination(oob[:oobn])
		if err != nil {
			return err
		}
		value := receipt{*id, destination.String(), strings.TrimSuffix(string(buf[:n]), "\n")}
		record(value)
		// Reply from the original destination, including its port. Sending from
		// the wildcard TPROXY listener would use the proxy port as the source.
		dialer := net.Dialer{LocalAddr: destination, Control: socketControl(*transparent)}
		reply, err := dialer.Dial(*network, peer.String())
		if err != nil {
			return err
		}
		// The connected reply socket can receive the next request before the
		// wildcard listener. Keep it open and serve that tuple; closing it
		// after sending a reply can discard the client's next datagram.
		go func(reply net.Conn, value receipt) {
			defer reply.Close()
			encoder := json.NewEncoder(reply)
			buf := make([]byte, 4096)
			for {
				_ = reply.SetDeadline(time.Now().Add(5 * time.Second))
				if err := encoder.Encode(value); err != nil {
					return
				}
				n, err := reply.Read(buf)
				if err != nil {
					return
				}
				value.Payload = strings.TrimSuffix(string(buf[:n]), "\n")
				record(value)
			}
		}(reply, value)
	}
}

func originalDestination(oob []byte) (*net.UDPAddr, error) {
	messages, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, err
	}
	for _, msg := range messages {
		if msg.Header.Level == unix.SOL_IP && msg.Header.Type == unix.IP_ORIGDSTADDR && len(msg.Data) >= 16 {
			return &net.UDPAddr{IP: net.IP(msg.Data[4:8]), Port: int(binary.BigEndian.Uint16(msg.Data[2:4]))}, nil
		}
		if msg.Header.Level == unix.SOL_IPV6 && msg.Header.Type == unix.IPV6_ORIGDSTADDR && len(msg.Data) >= 28 {
			return &net.UDPAddr{IP: net.IP(msg.Data[8:24]), Port: int(binary.BigEndian.Uint16(msg.Data[2:4]))}, nil
		}
	}
	return nil, errors.New("original UDP destination missing")
}

func client(args []string) error {
	flags := flag.NewFlagSet("client", flag.ExitOnError)
	network := flags.String("network", "tcp4", "transport")
	address := flags.String("address", "198.51.100.10:18080", "destination")
	expect := flags.String("expect", "proxy", "expected server identity, or drop")
	destination := flags.String("destination", "", "expected original destination")
	count := flags.Int("count", 2, "requests on the same connection")
	flags.Parse(args)
	conn, err := net.DialTimeout(*network, *address, 3*time.Second)
	if err != nil {
		return checkDrop(*expect, err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	decoder := json.NewDecoder(conn)
	for i := range *count {
		payload := fmt.Sprintf("cgtproxy-e2e-%d", i)
		if _, err := fmt.Fprintln(conn, payload); err != nil {
			return checkDrop(*expect, err)
		}
		var value receipt
		if strings.HasPrefix(*network, "udp") {
			buf := make([]byte, 4096)
			n, err := conn.Read(buf)
			if err != nil {
				return checkDrop(*expect, err)
			}
			err = json.Unmarshal(buf[:n], &value)
		} else {
			err = decoder.Decode(&value)
		}
		if err != nil {
			return checkDrop(*expect, err)
		}
		if value.Server != *expect || value.Payload != payload || (*destination != "" && value.Destination != *destination) {
			return fmt.Errorf("unexpected receipt %+v; want server=%s destination=%s payload=%s", value, *expect, *destination, payload)
		}
	}
	fmt.Printf("PASS %s %s -> %s\n", *network, *address, *expect)
	return nil
}

func checkDrop(expect string, err error) error {
	var networkErr net.Error
	if expect == "drop" && ((errors.As(err, &networkErr) && networkErr.Timeout()) || errors.Is(err, unix.EPERM)) {
		fmt.Println("PASS packet dropped")
		return nil
	}
	return err
}
