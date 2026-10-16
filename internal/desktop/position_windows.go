//go:build windows

package desktop

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var windowHandle atomic.Uintptr
var findWindowOnce sync.Once

// Resolve only this process's main window, never another Focus Lock instance.
func ownWindow() uintptr {
	findWindowOnce.Do(func() {
		callback := windows.NewCallback(func(hwnd, _ uintptr) uintptr {
			var pid uint32
			user32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
			if pid == uint32(os.Getpid()) {
				var title [256]uint16
				user32.NewProc("GetWindowTextW").Call(hwnd, uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
				if strings.HasPrefix(windows.UTF16ToString(title[:]), "Focus Lock") {
					windowHandle.Store(hwnd)
					return 0
				}
			}
			return 1
		})
		user32.NewProc("EnumWindows").Call(callback, 0)
	})
	return windowHandle.Load()
}

// Wails v2 WindowSetPosition adds the current monitor's origin even though
// WindowGetPosition returns absolute pixels. Use absolute Win32 coordinates so
// restoring saved bounds on another monitor does not add its origin twice.
func SetAbsolutePosition(x, y int) error {
	hwnd := ownWindow()
	if hwnd == 0 {
		return fmt.Errorf("Focus Lock window is unavailable")
	}
	ok, _, err := user32.NewProc("SetWindowPos").Call(hwnd, 0, uintptr(x), uintptr(y), 0, 0, 0x0001|0x0004|0x0010) // NOSIZE | NOZORDER | NOACTIVATE
	if ok == 0 {
		return fmt.Errorf("could not move window: %w", err)
	}
	return nil
}
