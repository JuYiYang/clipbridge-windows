package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerURL                   string        `json:"serverURL"`
	Token                       string        `json:"token"`
	SyncInterval                time.Duration `json:"-"`
	ClipboardPollInterval       time.Duration `json:"-"`
	ConfigPath                  string        `json:"-"`
	SyncIntervalSeconds         int           `json:"syncIntervalSeconds"`
	ClipboardPollIntervalMillis int           `json:"clipboardPollIntervalMillis"`
	StatePath                   string        `json:"statePath"`
}

func Load(path string) (Config, error) {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return Config{}, err
		}
		path = defaultPath
	}

	cfg := defaultConfig()
	bytes, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}
	if err == nil {
		if err := json.Unmarshal(bytes, &cfg); err != nil {
			return Config{}, fmt.Errorf("read config %s: %w", path, err)
		}
	}

	applyEnv(&cfg)
	Normalize(&cfg)
	cfg.ConfigPath = path
	return cfg, nil
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ClipBridge", "config.json"), nil
}

func Save(path string, cfg Config) error {
	if path == "" {
		if cfg.ConfigPath != "" {
			path = cfg.ConfigPath
		} else {
			defaultPath, err := DefaultPath()
			if err != nil {
				return err
			}
			path = defaultPath
		}
	}

	Normalize(&cfg)
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o600)
}

func WriteSample(path string) error {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return err
		}
		path = defaultPath
	}
	cfg := defaultConfig()
	cfg.ServerURL = "https://clipbridge-server.example.workers.dev"
	cfg.Token = ""
	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0o600)
}

func defaultConfig() Config {
	return Config{
		SyncIntervalSeconds:         300,
		ClipboardPollIntervalMillis: 500,
	}
}

func applyEnv(cfg *Config) {
	if value := strings.TrimSpace(os.Getenv("CLIPBRIDGE_SERVER_URL")); value != "" {
		cfg.ServerURL = value
	}
	if value := os.Getenv("CLIPBRIDGE_TOKEN"); value != "" {
		cfg.Token = value
	}
	if value := os.Getenv("CLIPBRIDGE_STATE_PATH"); value != "" {
		cfg.StatePath = value
	}
	if value := os.Getenv("CLIPBRIDGE_SYNC_INTERVAL_SECONDS"); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			cfg.SyncIntervalSeconds = seconds
		}
	}
	if value := os.Getenv("CLIPBRIDGE_CLIPBOARD_POLL_INTERVAL_MILLIS"); value != "" {
		if millis, err := strconv.Atoi(value); err == nil {
			cfg.ClipboardPollIntervalMillis = millis
		}
	}
}

func Normalize(cfg *Config) {
	cfg.ServerURL = strings.TrimRight(strings.TrimSpace(cfg.ServerURL), "/")
	if cfg.SyncIntervalSeconds < 15 {
		cfg.SyncIntervalSeconds = 15
	}
	if cfg.ClipboardPollIntervalMillis < 100 {
		cfg.ClipboardPollIntervalMillis = 100
	}
	cfg.SyncInterval = time.Duration(cfg.SyncIntervalSeconds) * time.Second
	cfg.ClipboardPollInterval = time.Duration(cfg.ClipboardPollIntervalMillis) * time.Millisecond
}
