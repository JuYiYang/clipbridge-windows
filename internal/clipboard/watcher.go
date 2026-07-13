package clipboard

import (
	"context"
	"time"
)

type Event struct {
	Text     string
	Captured time.Time
}

type Watcher interface {
	Events() <-chan Event
	Run(context.Context) error
}
