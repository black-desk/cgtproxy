package monitor_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	. "github.com/black-desk/deepin-network-proxy-manager/internal/core/monitor"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/watcher"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
	. "github.com/black-desk/deepin-network-proxy-manager/pkg/ginkgo-helper"
	. "github.com/black-desk/deepin-network-proxy-manager/pkg/gomega-helper"
	"github.com/fsnotify/fsnotify"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sourcegraph/conc/pool"
)

var _ = Describe("Cgroup monitor create with fake fsnotify.Watcher", func() {
	var (
		w               *watcher.Watcher
		cgroupEventChan chan *types.CgroupEvent
		ctx             context.Context
		monitor         *Monitor
	)

	BeforeEach(func() {
		w = &watcher.Watcher{
			Watcher: &fsnotify.Watcher{
				Events: make(chan fsnotify.Event),
				Errors: make(chan error),
			},
		}

		cgroupEventChan = make(chan *types.CgroupEvent)

		var cgroupEventChanIn chan<- *types.CgroupEvent
		cgroupEventChanIn = cgroupEventChan

		ctx = context.Background()

		var err error

		monitor, err = New(
			WithWatcher(w),
			WithCtx(ctx),
			WithOutput(cgroupEventChanIn),
		)
		Expect(err).To(Succeed())
	})

	ContextTable("receive %s", func(
		resultMsg string,
		events []fsnotify.Event, errs []error,
		expectResult []*types.CgroupEvent, expectErr error,
	) {
		var p *pool.ErrorPool

		BeforeEach(func() {
			p = new(pool.ErrorPool)

			p.Go(func() error {
				for i := range events {
					w.Events <- events[i]
				}
				close(w.Events)
				return nil
			})

			p.Go(func() error {
				// NOTE(black_desk): Errors from fsnotify is ignored for now.
				for i := range errs {
					w.Errors <- errs[i]
				}
				close(w.Errors)
				return nil
			})

			p.Go(monitor.Run)
		})

		AfterEach(func() {
			result := errors.Is(ctx.Err(), nil)
			Expect(result).To(BeTrue())
		})

		It(fmt.Sprintf("should %s", resultMsg), func() {
			var cgroupEvents []*types.CgroupEvent
			for cgroupEvent := range cgroupEventChan {
				cgroupEvents = append(cgroupEvents, cgroupEvent)
			}

			Expect(len(expectResult)).To(Equal(len(cgroupEvents)))

			for i := range cgroupEvents {
				Expect(*cgroupEvents[i]).To(Equal(*expectResult[i]))
			}

			err := p.Wait()
			Expect(err).To(MatchErr(expectErr))
		})
	},
		ContextTableEntry(
			"send a `New` event, and exit with no error",
			[]fsnotify.Event{{
				Name: "/test/path/1",
				Op:   fsnotify.Create,
			}},
			[]error{},
			[]*types.CgroupEvent{{
				Path:      "/test/path/1",
				EventType: types.CgroupEventTypeNew,
			}},
			nil,
		).WithFmt("a fsnotify.Event with fsnotify.Create"),
		ContextTableEntry(
			"send a `Delete` event, and exit with no error",
			[]fsnotify.Event{{
				Name: "/test/path/2",
				Op:   fsnotify.Remove,
			}},
			[]error{},
			[]*types.CgroupEvent{{
				Path:      "/test/path/2",
				EventType: types.CgroupEventTypeDelete,
			}},
			nil,
		).WithFmt("a fsnotify.Event with fsnotify.Delete"),
		ContextTableEntry(
			"send nothing, and exit with no error",
			[]fsnotify.Event{},
			[]error{},
			[]*types.CgroupEvent{},
			nil,
		).WithFmt("nothing"),
		ContextTableEntry(
			"send nothing, and exit with error",
			[]fsnotify.Event{{
				Name: "/test/path/3",
				Op:   fsnotify.Op(0),
			}},
			[]error{},
			[]*types.CgroupEvent{},
			new(ErrUnexpectFsEventOp),
		).WithFmt("invalid fsnotify.Event"),
	)
})

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}
