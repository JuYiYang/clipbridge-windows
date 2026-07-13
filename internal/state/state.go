package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Store struct {
	path string
}

type Data struct {
	DeviceID       string  `json:"deviceID"`
	LastPulledAt   float64 `json:"lastPulledAt"`
	LastUploadedID string  `json:"lastUploadedID"`
}

func NewStore(path string) Store {
	return Store{path: path}
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ClipBridge", "state.json"), nil
}

func (s Store) Load() (Data, error) {
	var data Data
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		data.DeviceID = newDeviceID()
		return data, nil
	}
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return data, err
	}
	if data.DeviceID == "" {
		data.DeviceID = newDeviceID()
	}
	return data, nil
}

func (s Store) Save(data Data) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, bytes, 0o600)
}

func newDeviceID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "windows-device"
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return "windows-" + hex.EncodeToString(bytes[:])
}
