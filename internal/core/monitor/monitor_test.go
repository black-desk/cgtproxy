package monitor_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/black-desk/deepin-network-proxy-manager/internal/config"
	. "github.com/black-desk/deepin-network-proxy-manager/internal/core/monitor"
	"github.com/black-desk/deepin-network-proxy-manager/internal/core/watcher"
	"github.com/black-desk/deepin-network-proxy-manager/internal/types"
	. "github.com/black-desk/lib/go/ginkgo-helper"
	. "github.com/black-desk/lib/go/gomega-helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sourcegraph/conc/pool"
	"github.com/tywkeene/go-fsevents"
	"golang.org/x/sys/unix"
)

var _ = Describe("Cgroup monitor create with fake fsevents.Watcher", Ordered, func() {
	var (
		w               *watcher.Watcher
		cgroupEventChan chan *types.CgroupEvent
		monitor         *Monitor
		tmpDir          string
		err             error
	)

	BeforeEach(func() {
		w = &watcher.Watcher{
			Watcher: &fsevents.Watcher{
				Events: make(chan *fsevents.FsEvent),
				Errors: make(chan error),
			},
		}

		cgroupEventChan = make(chan *types.CgroupEvent)

		var cgroupEventChanIn chan<- *types.CgroupEvent
		cgroupEventChanIn = cgroupEventChan

		tmpDir, err = os.MkdirTemp("/tmp", "*")
		Expect(err).To(Succeed())

		monitor, err = New(
			WithWatcher(w),
			WithOutput(cgroupEventChanIn),
			WithCgroupRoot(config.CgroupRoot(tmpDir)),
		)
		Expect(err).To(Succeed())
	})

	AfterEach(func() {
		err = os.RemoveAll(tmpDir)
		Expect(err).To(Succeed())
	})

	ContextTable("receive %s", func(
		resultMsg string,
		events []*fsevents.FsEvent, errs []error,
		expectResult []*types.CgroupEvent, expectErr error,
	) {
		var p *pool.ContextPool

		BeforeEach(func() {
			p = pool.New().WithContext(context.Background()).WithFirstError().WithCancelOnError()

			p.Go(func(ctx context.Context) error {
				defer close(w.Events)
				for i := range events {
					select {
					case w.Events <- events[i]:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return nil
			})

			p.Go(func(ctx context.Context) error {
				defer close(w.Errors)
				// NOTE(black_desk): Errors from fsevents is ignored for now.
				for i := range errs {
					select {
					case w.Errors <- errs[i]:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return nil
			})

			p.Go(monitor.Run)
		})

		It(fmt.Sprintf("should %s", resultMsg), func() {
			var cgroupEvents []*types.CgroupEvent
			for cgroupEvent := range cgroupEventChan {
				cgroupEvents = append(cgroupEvents, cgroupEvent)
			}

			Expect(len(expectResult) + 1).To(Equal(len(cgroupEvents)))

			Expect(cgroupEvents[0].Path).To(Equal(tmpDir))

			for i := range cgroupEvents[1:] {
				Expect(*cgroupEvents[i+1]).To(Equal(*expectResult[i]))
			}

			err = p.Wait()
			Expect(err).To(MatchErr(expectErr))
		})
	},
		ContextTableEntry(
			"send a `New` event, and exit with no error",
			[]*fsevents.FsEvent{{
				Path:     "/test/path/1",
				RawEvent: &unix.InotifyEvent{Mask: fsevents.IsDir | fsevents.Create | fsevents.MovedTo},
			}},
			[]error{},
			[]*types.CgroupEvent{{
				Path:      "/test/path/1",
				EventType: types.CgroupEventTypeNew,
			}},
			nil,
		).WithFmt("a DirCreated fsevents.Event"),
		ContextTableEntry(
			"send a `Delete` event, and exit with no error",
			[]*fsevents.FsEvent{{
				Path:     "/test/path/2",
				RawEvent: &unix.InotifyEvent{Mask: fsevents.IsDir | fsevents.Delete | fsevents.MovedFrom},
			}},
			[]error{},
			[]*types.CgroupEvent{{
				Path:      "/test/path/2",
				EventType: types.CgroupEventTypeDelete,
			}},
			nil,
		).WithFmt("a DirDeleted fsevents.Event"),
		ContextTableEntry(
			"send nothing, and exit with no error",
			[]*fsevents.FsEvent{},
			[]error{},
			[]*types.CgroupEvent{},
			nil,
		).WithFmt("nothing"),
		ContextTableEntry(
			"send nothing, and exit with error",
			[]*fsevents.FsEvent{{
				Path:     "/test/path/3",
				RawEvent: &unix.InotifyEvent{},
			}},
			[]error{},
			[]*types.CgroupEvent{},
			new(ErrUnexpectFsEvent),
		).WithFmt("invalid fsnotify.Event"),
	)
})

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}
