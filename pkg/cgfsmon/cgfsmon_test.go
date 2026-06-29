// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgfsmon

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/black-desk/cgtproxy/pkg/cgtproxy/config"
	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

func TestCGroupFSMonitor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CGroupFSMonitor Suite")
}

// makeTree creates a throwaway cgroup-like directory tree under a temp dir.
//
//	root/
//	  a/
//	  b/
//	    c/
//	  file.txt        (a regular file, must be skipped by the walker)
func makeTree(root string) {
	Expect(os.MkdirAll(filepath.Join(root, "a"), 0o755)).To(Succeed())
	Expect(os.MkdirAll(filepath.Join(root, "b", "c"), 0o755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(root, "file.txt"), []byte("x"), 0o644)).To(Succeed())
}

func eventPaths(events []types.CGroupEvent) []string {
	ret := make([]string, 0, len(events))
	for i := range events {
		ret = append(ret, events[i].Path)
	}
	sort.Strings(ret)
	return ret
}

var _ = Describe("CGroupFSMonitor", func() {
	Describe("construction via New", func() {
		Context("with missing or invalid arguments", func() {
			It("should fail when no cgroup root is provided", func() {
				_, err := New()
				Expect(err).To(MatchError(ErrCGroupRootNotFound))
			})

			It("should fail when the cgroup root is empty", func() {
				_, err := New(WithCgroupRoot(""))
				Expect(err).To(MatchError(ErrCGroupRootNotFound))
			})

			It("should fail when a nil logger is supplied", func() {
				_, err := New(WithCgroupRoot("/tmp"), WithLogger(nil))
				Expect(err).To(MatchError(ErrLoggerMissing))
			})
		})

		Context("with valid arguments", func() {
			It("should construct a monitor with the default buffer size", func() {
				m, err := New(WithCgroupRoot("/tmp"))
				Expect(err).ToNot(HaveOccurred())
				Expect(m).ToNot(BeNil())
				Expect(cap(m.eventsIn)).To(Equal(1024))
			})

			It("should accept a custom logger", func() {
				log := zap.NewExample().Sugar()
				m, err := New(WithCgroupRoot("/tmp"), WithLogger(log))
				Expect(err).ToNot(HaveOccurred())
				Expect(m.log).To(BeIdenticalTo(log))
			})
		})

		Context("with the buffer size controlled by CGTPROXY_MONITOR_BUFFER_SIZE", func() {
			It("should use the configured buffer size", func() {
				os.Setenv("CGTPROXY_MONITOR_BUFFER_SIZE", "256")
				defer os.Unsetenv("CGTPROXY_MONITOR_BUFFER_SIZE")

				m, err := New(WithCgroupRoot("/tmp"))
				Expect(err).ToNot(HaveOccurred())
				Expect(cap(m.eventsIn)).To(Equal(256))
			})

			It("should fall back to the default for an invalid value", func() {
				os.Setenv("CGTPROXY_MONITOR_BUFFER_SIZE", "not-a-number")
				defer os.Unsetenv("CGTPROXY_MONITOR_BUFFER_SIZE")

				m, err := New(WithCgroupRoot("/tmp"))
				Expect(err).ToNot(HaveOccurred())
				Expect(cap(m.eventsIn)).To(Equal(1024))
			})

			It("should fall back to the default for a non-positive value", func() {
				os.Setenv("CGTPROXY_MONITOR_BUFFER_SIZE", "0")
				defer os.Unsetenv("CGTPROXY_MONITOR_BUFFER_SIZE")

				m, err := New(WithCgroupRoot("/tmp"))
				Expect(err).ToNot(HaveOccurred())
				Expect(cap(m.eventsIn)).To(Equal(1024))
			})
		})
	})

	Describe("walk", func() {
		var (
			root string
			m    *CGroupFSMonitor
		)

		BeforeEach(func() {
			var err error
			root, err = os.MkdirTemp("", "cgfsmon-test-*")
			Expect(err).ToNot(HaveOccurred())
			makeTree(root)

			m, err = New(WithCgroupRoot(config.CGroupRoot(root)))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(root)).To(Succeed())
		})

		It("should collect every subdirectory except the root itself", func() {
			events, err := m.walk(root)
			Expect(err).ToNot(HaveOccurred())

			Expect(eventPaths(events)).To(ConsistOf(
				filepath.Join(root, "a"),
				filepath.Join(root, "b"),
				filepath.Join(root, "b", "c"),
			))
		})

		It("should report every collected directory as a New event", func() {
			events, err := m.walk(root)
			Expect(err).ToNot(HaveOccurred())
			for i := range events {
				Expect(events[i].EventType).To(Equal(types.CgroupEventTypeNew))
			}
		})
	})

	Describe("send", func() {
		var m *CGroupFSMonitor

		BeforeEach(func() {
			var err error
			m, err = New(WithCgroupRoot("/tmp"))
			Expect(err).ToNot(HaveOccurred())
		})

		It("should forward events to the output channel", func() {
			ctx := context.Background()

			received := make(chan types.CGroupEvents, 1)
			go func() {
				received <- <-m.Events()
			}()

			in := types.CGroupEvents{Events: []types.CGroupEvent{{
				Path:      "/some/cgroup/",
				EventType: types.CgroupEventTypeNew,
			}}}
			err := m.send(ctx, in)
			Expect(err).ToNot(HaveOccurred())

			Eventually(received).Should(Receive(WithTransform(
				func(e types.CGroupEvents) []string { return eventPaths(e.Events) },
				Equal([]string{"/some/cgroup"}), // trailing slash trimmed
			)))
		})

		It("should abort when the context is already cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := m.send(ctx, types.CGroupEvents{Events: []types.CGroupEvent{{
				Path: "/some/cgroup", EventType: types.CgroupEventTypeNew,
			}}})
			Expect(err).To(MatchError(context.Canceled))
		})
	})
})
