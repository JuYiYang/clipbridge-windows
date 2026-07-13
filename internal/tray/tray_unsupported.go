//go:build !windows

package tray

import "context"

type Options struct {
	SettingsURL string
	OnSyncNow   func()
	OnQuit      func()
}

func Run(ctx context.Context, _ Options) error {
	<-ctx.Done()
	return nil
}
