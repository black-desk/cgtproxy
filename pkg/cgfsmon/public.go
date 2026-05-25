// SPDX-FileCopyrightText: 2025 Chen Linxuan <me@black-desk.cn>
//
// SPDX-License-Identifier: GPL-3.0-or-later

package cgfsmon

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/black-desk/cgtproxy/pkg/types"
	. "github.com/black-desk/lib/go/errwrap"
	"github.com/rjeczalik/notify"
)

func (w *CGroupFSMonitor) Events() <-chan types.CGroupEvents {
	return w.eventsOut
}

func (w *CGroupFSMonitor) RunCGroupMonitor(ctx context.Context) (err error) {
	defer Wrap(&err, "running filesystem watcher")
	defer close(w.eventsOut)
	defer notify.Stop(w.eventsIn)
	defer close(w.eventsIn)

	// FIXME(workaround): during system startup, transient service cgroups
	// (e.g. lm-sensors.service) may disappear while rjeczalik/notify's
	// internal AddDir traverses the tree. The library does not tolerate
	// this (see its TODO in node.go AddDir). Retrying gives the system
	// time to settle.
	const maxRetries = 3
	for i := range maxRetries {
		err = notify.Watch(string(w.root)+"/...", w.eventsIn, notify.Create, notify.Remove)
		if err == nil {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return
		}
		w.log.Infow("notify.Watch failed, retrying...",
			"attempt", i+1,
			"error", err,
		)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	if err != nil {
		return
	}

	w.log.Info("Going through cgroupfs first time...")
	var events types.CGroupEvents
	events.Events, err = w.walk(string(w.root))
	w.log.Info("Going through cgroupfs first time...Done.")
	err = w.send(ctx, events)
	if err != nil {
		return
	}

LOOP:
	for {
		select {
		case <-ctx.Done():
			break LOOP
		case eventInfo := <-w.eventsIn:

			event := eventInfo.Event()
			path := eventInfo.Path()

			w.log.Debugw(
				"New filesystem notify arrived.",
				"event", event.String(),
				"path", path,
			)
			eventType := types.CgroupEventTypeNew
			if event == notify.Remove {
				eventType = types.CgroupEventTypeDelete
			}

			err = w.send(ctx, types.CGroupEvents{Events: []types.CGroupEvent{{
				Path:      path,
				EventType: eventType,
			}}})
		}
	}

	<-ctx.Done()
	err = context.Cause(ctx)
	if err != nil {
		return
	}

	return
}
