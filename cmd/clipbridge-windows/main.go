package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/JuYiYang/clipbridge-windows/internal/app"
	"github.com/JuYiYang/clipbridge-windows/internal/clipboard"
	"github.com/JuYiYang/clipbridge-windows/internal/config"
	"github.com/JuYiYang/clipbridge-windows/internal/settingsweb"
	"github.com/JuYiYang/clipbridge-windows/internal/state"
	"github.com/JuYiYang/clipbridge-windows/internal/syncclient"
	"github.com/JuYiYang/clipbridge-windows/internal/tray"
)

func main() {
	var configPath string
	var writeSampleConfig bool
	flag.StringVar(&configPath, "config", "", "Path to config.json")
	flag.BoolVar(&writeSampleConfig, "write-sample-config", false, "Write a sample config file and exit")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if writeSampleConfig {
		if err := config.WriteSample(configPath); err != nil {
			fatal(logger, err)
		}
		path, _ := config.DefaultPath()
		if configPath != "" {
			path = configPath
		}
		logger.Info("sample config written", "path", path)
		return
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		fatal(logger, err)
	}

	statePath := cfg.StatePath
	if statePath == "" {
		statePath, err = state.DefaultPath()
		if err != nil {
			fatal(logger, err)
		}
	}

	stateStore := state.NewStore(statePath)
	stateData, err := stateStore.Load()
	if err != nil {
		fatal(logger, err)
	}
	if err := stateStore.Save(stateData); err != nil {
		fatal(logger, err)
	}

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	ctx, cancel := context.WithCancel(signalCtx)
	defer cancel()

	client := syncclient.New(cfg.ServerURL, cfg.Token, stateData.DeviceID)
	watcher := clipboard.NewPollingWatcher(cfg.ClipboardPollInterval)
	agent := app.New(cfg, stateData, stateStore, client, watcher, logger)

	settingsServer := settingsweb.New(agent, cfg.ConfigPath)
	if err := settingsServer.Start(); err != nil {
		fatal(logger, err)
	}
	defer settingsServer.Shutdown(context.Background())

	go func() {
		if err := tray.Run(ctx, tray.Options{
			SettingsURL: settingsServer.URL(),
			OnSyncNow:   agent.SyncNow,
			OnQuit:      cancel,
		}); err != nil {
			logger.Warn("tray stopped", "error", err)
			cancel()
		}
	}()

	if err := agent.Run(ctx); err != nil {
		fatal(logger, err)
	}
}

func fatal(logger *slog.Logger, err error) {
	logger.Error("clipbridge windows stopped", "error", err)
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
