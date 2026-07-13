# ClipBridge Windows

ClipBridge Windows is a lightweight Windows clipboard sync agent.

It does not replace the native Windows clipboard history UI. The first version
runs as a small tray/foreground process that:

- watches new text copied while the agent is running,
- deduplicates repeated clipboard values,
- uploads text records to ClipBridge Server,
- periodically pulls the cloud cursor so records from other devices are visible
  on the server admin page.

The current implementation is intentionally small: Go, no Tauri, no Electron,
no UI framework.

## Status

Implemented:

- Go CLI agent
- native Windows tray menu
- local settings page opened from the tray
- Windows clipboard text polling through Win32 APIs
- cloud item restore into the Windows clipboard
- tray and executable icon resources
- Windows executable version metadata
- Stable local device ID
- Local JSON config and state files
- `POST /v1/clipboard/items`
- `GET /v1/clipboard/items?since=...`
- Bearer token auth via `CLIPBRIDGE_TOKEN`

Planned:

- background startup
- installer
- Windows Credential Manager or DPAPI for token storage
- images, HTML, RTF, and files
- end-to-end encryption envelopes

## Build

From this repository:

```sh
python3 scripts/generate_windows_resources.py
(cd cmd/clipbridge-windows && goversioninfo -64 -icon ../../assets/clipbridge.ico -o rsrc_windows_amd64.syso versioninfo.json)
go test ./...
GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui" -o dist/clipbridge-windows.exe ./cmd/clipbridge-windows
```

The clipboard watcher only runs on Windows. On macOS/Linux the binary will start
but exit with an unsupported-platform error.

`assets/clipbridge.ico` is generated from the macOS `StatusBarMenuImage`
asset. `cmd/clipbridge-windows/rsrc_windows_amd64.syso` embeds the tray/app icon
and Windows file properties into the executable.

## Configure

Launch the app and use the tray menu:

1. Right-click the ClipBridge tray icon.
2. Choose `设置...`.
3. Fill in Server URL, Access Token, and sync interval.
4. Click Save.

The settings UI is served from `127.0.0.1` while the app is running. It mirrors
the macOS Cloud settings pane: connection, token, schedule, status, and device
ID.

You can also write a sample config manually:

```powershell
clipbridge-windows.exe -write-sample-config
```

Default config path on Windows:

```text
%APPDATA%\ClipBridge\config.json
```

Example:

```json
{
  "serverURL": "https://clipbridge-server.example.workers.dev",
  "token": "",
  "syncIntervalSeconds": 300,
  "clipboardPollIntervalMillis": 500,
  "statePath": ""
}
```

If the server has `CLIPBRIDGE_TOKEN` configured, put the same token in `token`.

Environment variables override the config file:

- `CLIPBRIDGE_SERVER_URL`
- `CLIPBRIDGE_TOKEN`
- `CLIPBRIDGE_STATE_PATH`
- `CLIPBRIDGE_SYNC_INTERVAL_SECONDS`
- `CLIPBRIDGE_CLIPBOARD_POLL_INTERVAL_MILLIS`

## Run

```powershell
clipbridge-windows.exe
```

Or specify a config file:

```powershell
clipbridge-windows.exe -config C:\Users\you\AppData\Roaming\ClipBridge\config.json
```

The agent logs uploads and cloud pulls to stdout. Keep it running while copying
text that should be reported to ClipBridge Server.

Tray actions:

- Left-click the tray icon to open the local settings page.
- Right-click the tray icon to show the tray menu.
- `设置...` opens the local settings page.
- `立即同步` triggers a cloud pull immediately.
- `退出` stops the agent.
