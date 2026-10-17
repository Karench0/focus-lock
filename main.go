//go:build windows

package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/Karench0/focus-lock/internal/api"
	"github.com/Karench0/focus-lock/internal/applications"
	"github.com/Karench0/focus-lock/internal/bridge"
	"github.com/Karench0/focus-lock/internal/desktop"
	"github.com/Karench0/focus-lock/internal/model"
	"github.com/Karench0/focus-lock/internal/storage"
	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	winapi "golang.org/x/sys/windows"
)

//go:embed all:ui/dist
var assets embed.FS

//go:embed build/windows/icon.ico
var trayIcon []byte

type App struct {
	ctx       context.Context
	service   *api.Service
	dir       string
	quitting  atomic.Bool
	trayReady atomic.Bool
	ctxMu     sync.RWMutex
	window    *desktop.TimerWindow
}

func (a *App) context() context.Context { a.ctxMu.RLock(); defer a.ctxMu.RUnlock(); return a.ctx }
func (a *App) startup(ctx context.Context) {
	a.window = desktop.NewTimerWindow(nativeWindow{ctx})
	a.ctxMu.Lock()
	a.ctx = ctx
	a.ctxMu.Unlock()
}
func (a *App) domReady(ctx context.Context) {
	a.service.SetNotify(func() {
		_ = runtime.SendNotification(ctx, runtime.NotificationOptions{Title: "Focus Lock", Body: "Сессия завершена / Session complete. Ограничения сняты / Restrictions lifted."})
	})
	go systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTooltip("Focus Lock")
		show := systray.AddMenuItem("Открыть / Open Focus Lock", "")
		systray.AddSeparator()
		quit := systray.AddMenuItem("Выход / Quit", "")
		a.trayReady.Store(true)
		go func() {
			for {
				select {
				case <-show.ClickedCh:
					a.showFullWindow()
				case <-quit.ClickedCh:
					a.showFullWindow()
					if a.service.Active() {
						runtime.EventsEmit(ctx, "close-request")
					} else {
						a.quitting.Store(true)
						runtime.Quit(ctx)
					}
				}
			}
		}()
	}, func() {})
}
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting.Load() {
		return false
	}
	if a.service.Active() {
		// WM_CLOSE can invoke this on the UI thread. Return immediately before
		// calling window APIs which may synchronously dispatch onto that thread.
		go func() {
			a.showFullWindow()
			runtime.EventsEmit(ctx, "close-request")
		}()
		return true
	}
	return false
}
func (a *App) shutdown(context.Context) {
	a.service.Close()
	if a.trayReady.Load() {
		systray.Quit()
	}
}

// Call is a deliberately narrow Wails boundary. Every mutation is validated by
// the Go service; browser JavaScript cannot set timer state or unlock settings.
func (a *App) Call(command, payload string) (string, error) {
	var result any = true
	decode := func(v any) error { return json.Unmarshal([]byte(payload), v) }
	var err error
	switch command {
	case "state":
		result, err = a.service.State()
	case "discover":
		result = applications.Discover()
	case "saveProfile":
		var p model.Profile
		if err = decode(&p); err == nil {
			err = a.service.SaveProfile(p)
		}
	case "deleteProfile":
		var p struct {
			ID string `json:"id"`
		}
		if err = decode(&p); err == nil {
			err = a.service.DeleteProfile(p.ID)
		}
	case "saveSettings":
		var s model.Settings
		if err = decode(&s); err == nil {
			err = a.service.SaveSettings(s)
		}
	case "preflight":
		var p struct {
			ID string `json:"id"`
		}
		if err = decode(&p); err == nil {
			result, err = a.service.Preflight(p.ID)
		}
	case "start":
		var p struct {
			ID      string `json:"id"`
			Minutes int    `json:"minutes"`
		}
		if err = decode(&p); err == nil {
			err = a.service.Start(p.ID, p.Minutes)
		}
	case "requestExit":
		err = a.service.RequestExit()
	case "prepareStorozhenko":
		result, err = a.service.PrepareStorozhenko()
	case "startStorozhenko":
		var p struct {
			Token     string `json:"token"`
			Self      string `json:"self"`
			Confirmed bool   `json:"confirmed"`
		}
		if err = decode(&p); err == nil {
			err = a.service.StartStorozhenko(p.Token, p.Self, p.Confirmed)
		}
	case "kotlinQuestion":
		result, err = a.service.KotlinQuestion()
	case "answerKotlin":
		var p struct {
			Token  string `json:"token"`
			Option *int   `json:"option"`
		}
		if err = decode(&p); err == nil {
			if p.Option == nil {
				return "", fmt.Errorf("select an answer")
			}
			var correct bool
			correct, err = a.service.AnswerKotlin(p.Token, *p.Option)
			result = map[string]bool{"correct": correct}
		}
	case "openKotlinDocs":
		if a.context() == nil {
			return "", fmt.Errorf("requires desktop app")
		}
		err = desktop.OpenKotlinDocs()
	case "cancelExit":
		err = a.service.CancelExit()
	case "pickApplication":
		if a.context() == nil {
			return "", fmt.Errorf("file picker requires the desktop application")
		}
		var path string
		path, err = runtime.OpenFileDialog(a.context(), runtime.OpenDialogOptions{Title: "Выберите приложение / Select application", Filters: []runtime.FileFilter{{DisplayName: "Windows executable (*.exe)", Pattern: "*.exe"}}})
		result = nil
		if err == nil && path != "" {
			path, err = applications.Canonical(path)
			if err == nil {
				result = model.Application{Name: filepath.Base(path), Path: path, Icon: applications.Icon(path)}
			}
		}
	case "registerBridge":
		if a.service.Active() {
			return "", fmt.Errorf("session is active")
		}
		err = bridge.Register(filepath.Join(a.dir, "FocusLock-host.exe"))
	case "openExtension":
		if a.context() == nil {
			return "", fmt.Errorf("requires desktop app")
		}
		err = desktop.OpenExtensionFolder(a.dir)
	case "windowMode":
		var p struct {
			Compact       bool `json:"compact"`
			ReducedMotion bool `json:"reducedMotion"`
		}
		if err = decode(&p); err == nil {
			err = a.setCompact(p.Compact, !p.ReducedMotion)
		}
	case "windowPosition":
		ctx := a.context()
		if ctx == nil {
			return "", fmt.Errorf("requires desktop app")
		}
		x, y := runtime.WindowGetPosition(ctx)
		result = map[string]int{"x": x, "y": y}
	case "moveWindow":
		if a.context() == nil {
			return "", fmt.Errorf("requires desktop app")
		}
		var p struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err = decode(&p); err == nil {
			if p.X < -100000 || p.X > 100000 || p.Y < -100000 || p.Y > 100000 {
				return "", fmt.Errorf("invalid window position")
			}
			err = desktop.SetAbsolutePosition(p.X, p.Y)
		}
	case "minimise", "maximise":
		ctx := a.context()
		if ctx == nil {
			return "", fmt.Errorf("requires desktop app")
		}
		if command == "minimise" {
			runtime.WindowMinimise(ctx)
		} else {
			runtime.WindowToggleMaximise(ctx)
		}
	case "tray":
		if a.context() == nil {
			return "", fmt.Errorf("requires desktop app")
		}
		if !a.trayReady.Load() {
			return "", fmt.Errorf("tray is not ready; minimize the window instead")
		}
		runtime.WindowHide(a.context())
	case "quit":
		if a.context() == nil {
			return "", fmt.Errorf("requires desktop app")
		}
		if a.service.Active() {
			runtime.EventsEmit(a.context(), "close-request")
		} else {
			a.quitting.Store(true)
			runtime.Quit(a.context())
		}
	case "clearError":
		a.service.ClearError()
	default:
		return "", fmt.Errorf("unknown command")
	}
	if err != nil {
		return "", err
	}
	b, err := json.Marshal(result)
	return string(b), err
}
func main() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	dir := filepath.Dir(exe)
	preview := len(os.Args) > 1 && os.Args[1] == "--preview"
	windowPreview := len(os.Args) > 1 && os.Args[1] == "--window-preview"
	if !preview && !windowPreview {
		user, e := winapi.GetCurrentProcessToken().GetTokenUser()
		if e != nil {
			log.Fatal(e)
		}
		name, _ := winapi.UTF16PtrFromString(`Local\FocusLock-` + user.User.Sid.String())
		guard, e := winapi.CreateMutex(nil, false, name)
		if e != nil {
			log.Fatal("Focus Lock is already running, or its instance lock is unavailable: ", e)
		}
		defer winapi.CloseHandle(guard)
	}
	dataDir := os.Getenv("FOCUS_LOCK_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "FocusLock")
	}
	if err = os.MkdirAll(dataDir, 0700); err != nil {
		log.Fatal(err)
	}
	store, err := storage.Open(filepath.Join(dataDir, "focus-lock.db"))
	if err != nil {
		log.Fatal(err)
	}
	svc, err := api.New(store, filepath.Join(dir, "FocusLock-host.exe"), preview || windowPreview)
	if err != nil {
		_ = store.Close()
		log.Fatal(err)
	}
	app := &App{service: svc, dir: dir}
	if preview {
		defer svc.Close()
		if err = app.preview(); err != nil {
			log.Fatal(err)
		}
		return
	}
	title := "Focus Lock"
	if windowPreview {
		title = "Focus Lock · Window preview"
	}
	err = wails.Run(&options.App{Title: title, Frameless: true, Width: 1220, Height: 850, MinWidth: 980, MinHeight: 730, BackgroundColour: options.NewRGB(13, 15, 18), AssetServer: &assetserver.Options{Assets: assets}, OnStartup: app.startup, OnDomReady: app.domReady, OnBeforeClose: app.beforeClose, OnShutdown: app.shutdown, Bind: []interface{}{app}, Windows: &windows.Options{Theme: windows.Dark, DisableWindowIcon: false, ResizeDebounceMS: 0}})
	if err != nil {
		svc.Close()
		log.Fatal(err)
	}
}
