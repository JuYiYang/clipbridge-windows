package settingsweb

import (
	"context"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/JuYiYang/clipbridge-windows/internal/app"
	"github.com/JuYiYang/clipbridge-windows/internal/config"
)

type Server struct {
	agent      *app.App
	configPath string
	server     *http.Server
	url        string
}

type pageData struct {
	Status app.Status
	Saved  bool
	Error  string
}

func New(agent *app.App, configPath string) *Server {
	return &Server{
		agent:      agent,
		configPath: configPath,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/save", s.handleSave)
	mux.HandleFunc("/sync", s.handleSync)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	s.url = "http://" + listener.Addr().String()
	s.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = s.server.Serve(listener)
	}()
	return nil
}

func (s *Server) URL() string {
	return s.url
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := pageData{
		Status: s.agent.Status(),
		Saved:  r.URL.Query().Get("saved") == "1",
		Error:  r.URL.Query().Get("error"),
	}
	render(w, data)
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		redirectError(w, r, err)
		return
	}

	status := s.agent.Status()
	cfg := status.Config
	cfg.ServerURL = r.FormValue("serverURL")
	cfg.Token = r.FormValue("token")
	cfg.SyncIntervalSeconds = intFormValue(r, "syncIntervalSeconds", cfg.SyncIntervalSeconds)
	cfg.ClipboardPollIntervalMillis = intFormValue(r, "clipboardPollIntervalMillis", cfg.ClipboardPollIntervalMillis)

	if err := config.Save(s.configPath, cfg); err != nil {
		redirectError(w, r, err)
		return
	}

	config.Normalize(&cfg)
	s.agent.UpdateConfig(cfg)
	http.Redirect(w, r, "/?saved=1", http.StatusSeeOther)
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.agent.SyncNow()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func redirectError(w http.ResponseWriter, r *http.Request, err error) {
	http.Redirect(w, r, "/?error="+template.URLQueryEscaper(err.Error()), http.StatusSeeOther)
}

func intFormValue(r *http.Request, name string, fallback int) int {
	value, err := strconv.Atoi(r.FormValue(name))
	if err != nil {
		return fallback
	}
	return value
}

func render(w http.ResponseWriter, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplate.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("render settings: %v", err), http.StatusInternalServerError)
	}
}

var pageTemplate = template.Must(template.New("settings").Funcs(template.FuncMap{
	"formatTime": func(value time.Time) string {
		if value.IsZero() {
			return "-"
		}
		return value.Local().Format("2006-01-02 15:04:05")
	},
}).Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ClipBridge 云端</title>
  <style>
    :root { color-scheme: light dark; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    body { margin: 0; background: Canvas; color: CanvasText; }
    main { max-width: 760px; margin: 0 auto; padding: 32px 22px; }
    header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 22px; }
    h1 { margin: 0; font-size: 28px; }
    .subtitle, .hint, .status { color: color-mix(in srgb, CanvasText 58%, transparent); }
    form, section { border: 1px solid color-mix(in srgb, CanvasText 14%, transparent); border-radius: 8px; padding: 18px; margin-bottom: 14px; background: color-mix(in srgb, CanvasText 3%, transparent); }
    .section-title { font-weight: 700; margin-bottom: 10px; }
    label { display: grid; gap: 7px; margin: 12px 0; font-size: 13px; font-weight: 650; }
    input, select, button { min-height: 34px; border-radius: 7px; border: 1px solid color-mix(in srgb, CanvasText 18%, transparent); background: Canvas; color: CanvasText; padding: 0 10px; font: inherit; }
    input[type=password], input[type=text] { width: min(100%, 680px); box-sizing: border-box; }
    .actions { display: flex; gap: 10px; flex-wrap: wrap; margin-top: 16px; }
    button { cursor: pointer; font-weight: 700; }
    .primary { background: #0a84ff; color: white; border-color: #0a84ff; }
    .message { padding: 10px 12px; border-radius: 7px; margin-bottom: 14px; }
    .ok { background: color-mix(in srgb, #34c759 20%, transparent); }
    .error { background: color-mix(in srgb, #ff3b30 18%, transparent); }
    dl { display: grid; grid-template-columns: 150px 1fr; gap: 8px 12px; margin: 0; }
    dt { color: color-mix(in srgb, CanvasText 58%, transparent); }
    dd { margin: 0; overflow-wrap: anywhere; }
    code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; }
  </style>
</head>
<body>
  <main>
    <header>
      <div>
        <h1>云端</h1>
        <div class="subtitle">ClipBridge Windows 同步设置</div>
      </div>
      <form method="post" action="/sync" style="border:0;padding:0;margin:0;background:transparent">
        <button type="submit">立即同步</button>
      </form>
    </header>

    {{if .Saved}}<div class="message ok">设置已保存，并已触发同步。</div>{{end}}
    {{if .Error}}<div class="message error">{{.Error}}</div>{{end}}

    <form method="post" action="/save">
      <div class="section-title">连接</div>
      <label>服务地址
        <input name="serverURL" type="text" value="{{.Status.Config.ServerURL}}" placeholder="https://clipbridge-server.example.workers.dev">
      </label>
      <label>访问 Token
        <input name="token" type="password" value="{{.Status.Config.Token}}">
      </label>
      <p class="hint">Token 会作为 Bearer Token 发送给 ClipBridge 服务端。</p>

      <div class="section-title">同步时间</div>
      <label>同步间隔
        <select name="syncIntervalSeconds">
          {{$current := .Status.Config.SyncIntervalSeconds}}
          <option value="15" {{if eq $current 15}}selected{{end}}>15s</option>
          <option value="30" {{if eq $current 30}}selected{{end}}>30s</option>
          <option value="60" {{if eq $current 60}}selected{{end}}>1m</option>
          <option value="300" {{if eq $current 300}}selected{{end}}>5m</option>
          <option value="900" {{if eq $current 900}}selected{{end}}>15m</option>
          <option value="3600" {{if eq $current 3600}}selected{{end}}>1h</option>
        </select>
      </label>
      <label>剪贴板检测间隔 ms
        <input name="clipboardPollIntervalMillis" type="number" min="100" step="100" value="{{.Status.Config.ClipboardPollIntervalMillis}}">
      </label>
      <p class="hint">程序运行期间捕获到的新剪贴板内容仍会立即上传。</p>

      <div class="actions">
        <button class="primary" type="submit">保存</button>
      </div>
    </form>

    <section>
      <div class="section-title">状态</div>
      <dl>
        <dt>设备 ID</dt><dd><code>{{.Status.DeviceID}}</code></dd>
        <dt>最近同步</dt><dd>{{formatTime .Status.LastSyncedAt}}</dd>
        <dt>最近上传</dt><dd>{{formatTime .Status.LastUploadedAt}}</dd>
        <dt>上传标题</dt><dd>{{if .Status.LastUploaded}}{{.Status.LastUploaded}}{{else}}-{{end}}</dd>
        <dt>错误</dt><dd>{{if .Status.LastError}}{{.Status.LastError}}{{else}}-{{end}}</dd>
      </dl>
    </section>
  </main>
</body>
</html>`))
