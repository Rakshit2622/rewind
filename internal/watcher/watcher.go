package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"rewind/internal/diff"
	"rewind/internal/metadata"
	"rewind/internal/snapshot"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type WatchEvent struct {
	Type string // "saved", "save_error", "fs_error", "skipped", "initial save", ""
	Err  error
	Data string
}

func sendEvent(ch chan<- WatchEvent, evt WatchEvent) {
	if ch == nil {
		return
	}
	select {
	case ch <- evt:
	default:
	}
}

func isTempFile(name string) bool {
	base := filepath.Base(name)
	return strings.HasSuffix(base, "~") ||
		strings.HasSuffix(base, ".swp") ||
		strings.HasSuffix(base, ".swx") ||
		strings.HasSuffix(base, ".tmp") ||
		strings.HasPrefix(base, ".#")
}

// Watch starts monitoring filePath for changes and automatically saves a
// snapshot after the file has been stable for debounceMs milliseconds.
// Blocks until ctx is cancelled (Ctrl+C).
func Watch(
	ctx context.Context,
	filePath string,
	debounceMs int,
	eventCh chan<- WatchEvent,
	with_preview bool,
) error {
	err := metadata.ValidateFile(filePath)
	if err != nil {
		return err
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	if err := w.Add(filePath); err != nil {
		return err
	}
	var timer *time.Timer
	var lastContent string
	if data, err := os.ReadFile(filePath); err == nil {
		lastContent = string(data)
	}
	str := fmt.Sprintf("Initial snapshot @ %s", time.Now().Format(time.RFC3339))
	if err := snapshot.Save(filePath, str); err != nil {
		sendEvent(eventCh, WatchEvent{Type: "save_error", Err: err})
	}
	sendEvent(eventCh, WatchEvent{Type: "initial_save"})
	for {
		select {
		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			eventPath := filepath.Clean(event.Name)
			targetPath := filepath.Clean(filePath)
			if event.Op&fsnotify.Write == fsnotify.Write && eventPath == targetPath {
				if timer != nil {
					timer.Stop()
				}

				timer = startDebounce(filePath, debounceMs, eventCh, &lastContent, with_preview)
				continue
			}
			if event.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
				if isTempFile(eventPath) {
					continue
				}
				if _, err := os.Stat(filePath); err == nil {

					if timer != nil {
						timer.Stop()
					}

					timer = startDebounce(filePath, debounceMs, eventCh, &lastContent, with_preview)
				}
			}
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}

			sendEvent(eventCh, WatchEvent{
				Type: "fs_error",
				Err:  err,
			})
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return nil
		}
	}
}

// startDebounce resets the debounce timer on each write event.
// When the timer fires it calls snapshot.Save with an auto-generated message.
func startDebounce(
	filePath string,
	debounceMs int,
	eventCh chan<- WatchEvent,
	lastContent *string,
	with_preview bool,
) *time.Timer {
	return time.AfterFunc(time.Duration(debounceMs)*time.Millisecond, func() {
		data, err := os.ReadFile(filePath)
		if err != nil {
			sendEvent(eventCh, WatchEvent{Type: "save_error", Err: err})
			return
		}
		current := string(data)
		if lastContent != nil && *lastContent == current {
			sendEvent(eventCh, WatchEvent{Type: "skipped"})
			return
		}
		msg := fmt.Sprintf("auto snapshot @ %s", time.Now().Format(time.RFC3339))
		if err := snapshot.Save(filePath, msg); err != nil {
			sendEvent(eventCh, WatchEvent{Type: "save_error", Err: err})
			return
		}
		if with_preview {
			diffOutput := diff.Compute(*lastContent, current)
			sendEvent(eventCh, WatchEvent{Type: "diff_preview", Data: diffOutput})
		}
		if lastContent != nil {
			*lastContent = current
		}
		sendEvent(eventCh, WatchEvent{Type: "saved"})
	})
}
