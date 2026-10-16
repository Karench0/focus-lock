//go:build windows

package main

import (
	"context"
	"fmt"

	"github.com/Karench0/focus-lock/internal/desktop"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type nativeWindow struct{ ctx context.Context }

func (w nativeWindow) Bounds() desktop.Bounds {
	x, y := runtime.WindowGetPosition(w.ctx)
	width, height := runtime.WindowGetSize(w.ctx)
	return desktop.Bounds{X: x, Y: y, Width: width, Height: height}
}
func (w nativeWindow) SetSize(x, y int) { runtime.WindowSetSize(w.ctx, x, y) }
func (w nativeWindow) SetPosition(x, y int) {
	if err := desktop.SetAbsolutePosition(x, y); err != nil {
		runtime.LogError(w.ctx, err.Error())
	}
}
func (w nativeWindow) SetMinSize(x, y int)    { runtime.WindowSetMinSize(w.ctx, x, y) }
func (w nativeWindow) SetMaxSize(x, y int)    { runtime.WindowSetMaxSize(w.ctx, x, y) }
func (w nativeWindow) SetAlwaysOnTop(on bool) { runtime.WindowSetAlwaysOnTop(w.ctx, on) }
func (w nativeWindow) IsMaximised() bool      { return runtime.WindowIsMaximised(w.ctx) }
func (w nativeWindow) Unmaximise()            { runtime.WindowUnmaximise(w.ctx) }
func (w nativeWindow) Maximise()              { runtime.WindowMaximise(w.ctx) }

func (a *App) setCompact(compact, animate bool) error {
	ctx := a.context()
	if ctx == nil {
		return fmt.Errorf("requires desktop app")
	}
	a.window.SetCompact(compact, animate)
	runtime.EventsEmit(ctx, "window-mode", compact)
	return nil
}

func (a *App) showFullWindow() {
	ctx := a.context()
	runtime.WindowShow(ctx)
	runtime.WindowUnminimise(ctx)
	_ = a.setCompact(false, true)
}
