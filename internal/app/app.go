package app

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/JuYiYang/clipbridge-windows/internal/clipboard"
	"github.com/JuYiYang/clipbridge-windows/internal/config"
	"github.com/JuYiYang/clipbridge-windows/internal/protocol"
	"github.com/JuYiYang/clipbridge-windows/internal/state"
	"github.com/JuYiYang/clipbridge-windows/internal/syncclient"
)

type App struct {
	cfg config.Config

	mu         sync.RWMutex
	state      state.Data
	stateStore state.Store
	client     syncclient.Client
	watcher    clipboard.Watcher
	logger     *slog.Logger

	lastError       string
	lastSyncedAt    time.Time
	lastUploadedAt  time.Time
	lastUploaded    string
	suppressedText  string
	suppressUntil   time.Time
	syncNowRequests chan struct{}
}

type Status struct {
	Config         config.Config
	DeviceID       string
	LastPulledAt   float64
	LastError      string
	LastSyncedAt   time.Time
	LastUploadedAt time.Time
	LastUploaded   string
}

func New(
	cfg config.Config,
	stateData state.Data,
	stateStore state.Store,
	client syncclient.Client,
	watcher clipboard.Watcher,
	logger *slog.Logger,
) *App {
	return &App{
		cfg:        cfg,
		state:      stateData,
		stateStore: stateStore,
		client:     client,
		watcher:    watcher,
		logger:     logger,

		syncNowRequests: make(chan struct{}, 1),
	}
}

func (a *App) Run(ctx context.Context) error {
	watcherDone := make(chan error, 1)
	go func() {
		watcherDone <- a.watcher.Run(ctx)
	}()

	pullTicker := time.NewTicker(a.currentConfig().SyncInterval)
	defer pullTicker.Stop()

	cfg := a.currentConfig()
	a.logger.Info("clipbridge windows started",
		"server", cfg.ServerURL,
		"deviceID", a.deviceID(),
		"syncInterval", cfg.SyncInterval.String(),
	)

	if err := a.pull(ctx); err != nil {
		a.logger.Warn("initial pull failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-watcherDone:
			if err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		case event, ok := <-a.watcher.Events():
			if !ok {
				return nil
			}
			if err := a.uploadText(ctx, event); err != nil {
				a.recordError(err)
				a.logger.Warn("upload failed", "error", err)
			}
		case <-a.syncNowRequests:
			if err := a.pull(ctx); err != nil {
				a.recordError(err)
				a.logger.Warn("manual sync failed", "error", err)
			}
		case <-pullTicker.C:
			if err := a.pull(ctx); err != nil {
				a.recordError(err)
				a.logger.Warn("pull failed", "error", err)
			}
			pullTicker.Reset(a.currentConfig().SyncInterval)
		}
	}
}

func (a *App) UpdateConfig(cfg config.Config) {
	config.Normalize(&cfg)

	a.mu.Lock()
	a.cfg = cfg
	a.client = syncclient.New(cfg.ServerURL, cfg.Token, a.state.DeviceID)
	a.mu.Unlock()

	a.SyncNow()
}

func (a *App) SyncNow() {
	select {
	case a.syncNowRequests <- struct{}{}:
	default:
	}
}

func (a *App) Status() Status {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return Status{
		Config:         a.cfg,
		DeviceID:       a.state.DeviceID,
		LastPulledAt:   a.state.LastPulledAt,
		LastError:      a.lastError,
		LastSyncedAt:   a.lastSyncedAt,
		LastUploadedAt: a.lastUploadedAt,
		LastUploaded:   a.lastUploaded,
	}
}

func (a *App) uploadText(ctx context.Context, event clipboard.Event) error {
	if a.shouldSuppressUpload(event.Text) {
		a.logger.Info("ignored clipboard event applied from cloud")
		return nil
	}

	if !a.isConfigured() {
		return nil
	}

	deviceID := a.deviceID()
	item := protocol.NewTextItem(deviceID, event.Text, event.Captured)
	a.mu.RLock()
	lastUploadedID := a.state.LastUploadedID
	a.mu.RUnlock()
	if item.ID == lastUploadedID {
		return nil
	}

	response, err := a.currentClient().PushItems(ctx, []protocol.ClipboardItem{item})
	if err != nil {
		return err
	}

	a.mu.Lock()
	a.state.LastUploadedID = item.ID
	if response.NextSince > a.state.LastPulledAt {
		a.state.LastPulledAt = response.NextSince
	}
	a.lastUploaded = item.Title
	a.lastUploadedAt = time.Now().UTC()
	a.lastSyncedAt = time.Now().UTC()
	a.lastError = ""
	stateData := a.state
	a.mu.Unlock()

	if err := a.stateStore.Save(stateData); err != nil {
		return err
	}

	a.logger.Info("uploaded clipboard text", "title", item.Title, "stored", response.Stored)
	return nil
}

func (a *App) pull(ctx context.Context) error {
	if !a.isConfigured() {
		return nil
	}

	a.mu.RLock()
	since := a.state.LastPulledAt
	deviceID := a.state.DeviceID
	a.mu.RUnlock()

	response, err := a.currentClient().PullItems(ctx, since)
	if err != nil {
		return err
	}

	remoteCount := 0
	var latestText string
	var latestCopiedAt time.Time
	for _, item := range response.Items {
		if item.SourceDeviceID != deviceID {
			remoteCount += 1
			text, ok := plainText(item)
			if ok && (latestCopiedAt.IsZero() || item.LastCopiedAt.After(latestCopiedAt)) {
				latestText = text
				latestCopiedAt = item.LastCopiedAt
			}
		}
	}

	if latestText != "" {
		if err := clipboard.WriteText(ctx, latestText); err != nil {
			return err
		}
		a.suppressCloudAppliedText(latestText)
	}

	a.mu.Lock()
	if response.NextSince != nil {
		a.state.LastPulledAt = *response.NextSince
	}
	a.lastSyncedAt = time.Now().UTC()
	a.lastError = ""
	stateData := a.state
	a.mu.Unlock()

	if response.NextSince != nil {
		if err := a.stateStore.Save(stateData); err != nil {
			return err
		}
	}

	if remoteCount > 0 {
		a.logger.Info("pulled remote clipboard records", "count", remoteCount, "applied", latestText != "")
	}
	return nil
}

func (a *App) currentConfig() config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

func (a *App) currentClient() syncclient.Client {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client
}

func (a *App) deviceID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state.DeviceID
}

func (a *App) recordError(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastError = err.Error()
}

func (a *App) isConfigured() bool {
	return a.currentConfig().ServerURL != ""
}

func (a *App) suppressCloudAppliedText(text string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.suppressedText = text
	a.suppressUntil = time.Now().Add(5 * time.Second)
}

func (a *App) shouldSuppressUpload(text string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.suppressedText == "" || text != a.suppressedText || time.Now().After(a.suppressUntil) {
		return false
	}
	a.suppressedText = ""
	a.suppressUntil = time.Time{}
	return true
}

func plainText(item protocol.ClipboardItem) (string, bool) {
	for _, content := range item.Contents {
		if content.Type != protocol.PlainTextType {
			continue
		}
		data, err := base64.StdEncoding.DecodeString(content.Value)
		if err != nil {
			return "", false
		}
		return string(data), true
	}
	return "", false
}
