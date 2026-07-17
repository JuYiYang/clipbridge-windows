# ClipBridge Windows Release

This repository builds the Windows tray sync agent.

## Local Build

From this repository:

```sh
python3 scripts/generate_windows_resources.py
cd cmd/clipbridge-windows
goversioninfo -64 -icon ../../assets/clipbridge.ico -o rsrc_windows_amd64.syso versioninfo.json
cd ../..
go test ./...
GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui" -o dist/clipbridge-windows.exe ./cmd/clipbridge-windows
```

Package the executable:

```sh
ditto -c -k --keepParent \
  dist/clipbridge-windows.exe \
  dist/clipbridge-windows-amd64.zip
```

Output:

```text
dist/clipbridge-windows.exe
dist/clipbridge-windows-amd64.zip
```

## GitHub Release Packaging

The workflow in `.github/workflows/release.yml` builds
`clipbridge-windows.exe`, packages it into `clipbridge-windows-amd64.zip`, and
uploads the zip as a workflow artifact.

When a tag matching `v*` is pushed, the workflow also creates or updates a
GitHub Release and uploads the zip as a release asset.

Example:

```sh
git tag v0.1.0
git push origin v0.1.0
```

## Runtime Configuration

Default config path on Windows:

```text
%APPDATA%\ClipBridge\config.json
```

Fields:

- `serverURL`: Cloudflare Worker URL.
- `token`: Same value as the Worker secret `CLIPBRIDGE_TOKEN`.
- `syncIntervalSeconds`: Pull interval for cloud records.
- `clipboardPollIntervalMillis`: Local clipboard polling interval.
- `statePath`: Optional override for local state.

The app currently stores a local device ID and sync cursor in:

```text
%APPDATA%\ClipBridge\state.json
```
