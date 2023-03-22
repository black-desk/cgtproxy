package demofswatch

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/black-desk/deepin-network-proxy-manager/internal/fswatch"
	"github.com/black-desk/deepin-network-proxy-manager/internal/log"
)

func (w *FsWatcherDemo) Start() (events <-chan *fswatch.FsEvent, err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf("Failed to start fswatch: %w", err)
		return
	}()

	w.lock.Lock()
	defer w.lock.Unlock()

	if w.channel != nil {
		err = fmt.Errorf("fswatch has been started.")
		return
	}

	channel := make(chan *fswatch.FsEvent)
	w.channel = channel
	go w.run()

	events = channel
	return
}

func (w *FsWatcherDemo) run() {

	var err error

	defer func() {
		if err != nil {
			w.sendErr(err)
		}
		close(w.channel)
	}()

	args := []string{"-x"}

	if w.recursive {
		args = append(args, "-r")
	}

	for i := range w.events {
		args = append(args, "--event", w.events[i])
	}

	args = append(args, w.path)

	log.Debug().Printf("fswatch args: %v", args)

	cmd := exec.CommandContext(w.ctx, "fswatch", args...)

	var out io.Reader
	if out, err = cmd.StdoutPipe(); err != nil {
		err = fmt.Errorf(
			"Failed to get stdout of process fswatch: %w",
			err,
		)
		return
	}

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf(
			"Failed to start process fswatch: %w",
			err,
		)
		return
	}

	reader := bufio.NewReader(out)
	for {
		var line string
		line, err = reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			err = nil
			return
		}
		if err != nil {
			err = fmt.Errorf(
				"Failed to read a string from stdout of process fswatch: %w",
				err,
			)
			return
		}

		var event *fswatch.FsEvent

		event, err = w.parseLine(line[0 : len(line)-1])
		if err != nil {
			return
		}

		err = w.sendEvent(event)
		if err != nil {
			return
		}
	}
}

func (w *FsWatcherDemo) sendErr(err error) {
	if err == nil {
		panic("This function should never be used to send nil.")
	}
	select {
	case w.channel <- &fswatch.FsEvent{
		Err: fmt.Errorf(
			"Error occurs while running fswatch: %w",
			err,
		),
	}:
	case <-w.ctx.Done():
	}
}

func (w *FsWatcherDemo) parseLine(line string) (event *fswatch.FsEvent, err error) {
	component := strings.Split(line, " ")
	if len(component) != 2 {
		err = fmt.Errorf(
			"Unexpected string read from process stdout of fswatch: %s",
			line,
		)
		return
	}

	var fsEventType fswatch.FsEventType

	switch component[1] {
	case "NoOp":
		fsEventType = fswatch.FsEventTypeNoOp
	case "PlatformSpecific":
		fsEventType = fswatch.FsEventTypePlatformSpecific
	case "Created":
		fsEventType = fswatch.FsEventTypeCreated
	case "Updated":
		fsEventType = fswatch.FsEventTypeUpdated
	case "Removed":
		fsEventType = fswatch.FsEventTypeRemoved
	case "Renamed":
		fsEventType = fswatch.FsEventTypeRenamed
	case "OwnerModified":
		fsEventType = fswatch.FsEventTypeOwnerModified
	case "AttributeModified":
		fsEventType = fswatch.FsEventTypeAttributeModified
	case "MovedFrom":
		fsEventType = fswatch.FsEventTypeMovedFrom
	case "MovedTo":
		fsEventType = fswatch.FsEventTypeMovedTo
	case "IsFile":
		fsEventType = fswatch.FsEventTypeIsFile
	case "IsDir":
		fsEventType = fswatch.FsEventTypeIsDir
	case "IsSymLink":
		fsEventType = fswatch.FsEventTypeIsSymLink
	case "Link":
		fsEventType = fswatch.FsEventTypeLink
	case "Overflow":
		fsEventType = fswatch.FsEventTypeOverflow
	default:
		err = fmt.Errorf("Unexpected fs event type: %s", component[1])
		return
	}

	event = &fswatch.FsEvent{
		Type: fsEventType,
		Path: component[0],
		Err:  nil,
	}

	return
}

func (w *FsWatcherDemo) sendEvent(event *fswatch.FsEvent) (err error) {
	defer func() {
		if err == nil {
			return
		}

		err = fmt.Errorf(
			"Failed to send event though channel: %w",
			err,
		)
	}()

	select {
	case w.channel <- event:
	case <-w.ctx.Done():
		err = context.Canceled
	}

	return
}
