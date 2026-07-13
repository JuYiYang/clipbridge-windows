//go:build !windows

package clipboard

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

type PollingWatcher struct {
	events chan Event
}

func NewPollingWatcher(_ time.Duration) *PollingWatcher {
	return &PollingWatcher{events: make(chan Event)}
}

func (w *PollingWatcher) Events() <-chan Event {
	return w.events
}

func (w *PollingWatcher) Run(context.Context) error {
	close(w.events)
	return fmt.Errorf("clipboard watcher is only available on Windows; current platform is %s", runtime.GOOS)
}

func WriteText(context.Context, string) error {
	return fmt.Errorf("clipboard writer is only available on Windows; current platform is %s", runtime.GOOS)
}
