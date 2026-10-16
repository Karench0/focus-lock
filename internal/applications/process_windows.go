//go:build windows

package applications

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var user32 = windows.NewLazySystemDLL("user32.dll")
var kernel32 = windows.NewLazySystemDLL("kernel32.dll")
var isCritical = kernel32.NewProc("IsProcessCritical")

var ErrProcessProtected = errors.New("process is protected")

// Protection can change between the snapshot and termination (including AV
// self-protection). An access denial is an exemption, never a reason to elevate.
func IsProtectedProcessError(err error) bool {
	return errors.Is(err, ErrProcessProtected) || errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD)
}

func protectedProcessName(name string) bool {
	switch strings.ToLower(name) {
	case "taskmgr.exe", "explorer.exe", "narrator.exe", "magnify.exe", "osk.exe", "nvda.exe", "jfw.exe",
		"msmpeng.exe", "nissrv.exe", "securityhealthservice.exe", "defendersessionhelper.exe",
		"focuslock.exe", "focuslock-host.exe", "msedgewebview2.exe":
		return true
	}
	return false
}

// Windows callbacks cannot be freed. Allocate one for the lifetime of the app,
// rather than leaking a callback on each enforcement snapshot.
var visibleMu sync.Mutex
var currentVisible map[uint32]bool
var enumVisibleCallback = syscall.NewCallback(func(hwnd, unused uintptr) uintptr {
	visible, _, _ := user32.NewProc("IsWindowVisible").Call(hwnd)
	if visible != 0 {
		var pid uint32
		user32.NewProc("GetWindowThreadProcessId").Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		currentVisible[pid] = true
	}
	return 1
})

func Canonical(path string) (string, error) {
	if !strings.EqualFold(filepath.Ext(path), ".exe") {
		return "", errors.New("select an .exe file")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs, err = filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	f, err := os.Open(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		return "", errors.New("executable is not a regular file")
	}
	var magic [2]byte
	if _, err = f.Read(magic[:]); err != nil || string(magic[:]) != "MZ" {
		return "", errors.New("file is not a Windows executable")
	}
	buf := make([]uint16, 32768)
	n, err := windows.GetFinalPathNameByHandle(windows.Handle(f.Fd()), &buf[0], uint32(len(buf)), 0)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(windows.UTF16ToString(buf[:n]), `\\?\`), nil
}

func visiblePIDs() map[uint32]bool {
	visibleMu.Lock()
	defer visibleMu.Unlock()
	out := map[uint32]bool{}
	currentVisible = out
	user32.NewProc("EnumWindows").Call(enumVisibleCallback, 0)
	currentVisible = nil
	return out
}
func Snapshot() ([]Process, error) {
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)
	visible := visiblePIDs()
	out := []Process{}
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	for err = windows.Process32First(h, &entry); err == nil; err = windows.Process32Next(h, &entry) {
		if p, e := Inspect(entry.ProcessID, entry.ParentProcessID); e == nil {
			p.Visible = visible[p.PID]
			out = append(out, p)
		}
	}
	if err != windows.ERROR_NO_MORE_FILES {
		return nil, err
	}
	return out, nil
}
func Inspect(pid, parent uint32) (Process, error) {
	p := Process{PID: pid, Parent: parent, Safe: true}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return p, err
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, 32768)
	n := uint32(len(buf))
	if err = windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return p, err
	}
	p.Path = windows.UTF16ToString(buf[:n])
	p.Name = filepath.Base(p.Path)
	var created, exited, kernel, user windows.Filetime
	if err = windows.GetProcessTimes(h, &created, &exited, &kernel, &user); err != nil {
		return p, err
	}
	p.Created = created.Nanoseconds()
	var sid, ourSession uint32
	if windows.ProcessIdToSessionId(pid, &sid) != nil || windows.ProcessIdToSessionId(uint32(os.Getpid()), &ourSession) != nil || sid != ourSession || sid == 0 {
		return p, nil
	}
	var tok windows.Token
	if windows.OpenProcessToken(h, windows.TOKEN_QUERY, &tok) != nil {
		return p, nil
	}
	defer tok.Close()
	u, err := tok.GetTokenUser()
	if err != nil {
		return p, nil
	}
	me, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil || !u.User.Sid.Equals(me.User.Sid) {
		return p, nil
	}
	var elevated, elevationSize uint32
	if windows.GetTokenInformation(tok, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &elevationSize) != nil || elevated != 0 {
		return p, nil
	}
	var uiAccess, returned uint32
	if windows.GetTokenInformation(tok, windows.TokenUIAccess, (*byte)(unsafe.Pointer(&uiAccess)), uint32(unsafe.Sizeof(uiAccess)), &returned) != nil || uiAccess != 0 {
		return p, nil
	}
	var critical int32
	ok, _, _ := isCritical.Call(uintptr(h), uintptr(unsafe.Pointer(&critical)))
	if ok == 0 || critical != 0 {
		return p, nil
	}
	path := strings.ToLower(p.Path)
	for _, dir := range securityDirectories() {
		if strings.HasPrefix(path, dir) {
			return p, nil
		}
	}
	for _, dir := range []string{os.Getenv("WINDIR"), filepath.Join(os.Getenv("ProgramFiles"), "Windows Defender"), filepath.Join(os.Getenv("ProgramFiles"), "Microsoft Defender"), filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "EdgeWebView")} {
		if dir != "" && strings.HasPrefix(path, strings.ToLower(filepath.Clean(dir))+`\`) {
			return p, nil
		}
	}
	// Fail safe for accessibility, security agents and the app's own runtime.
	if protectedProcessName(p.Name) {
		return p, nil
	}
	if pid == uint32(os.Getpid()) {
		return p, nil
	}
	// Query access does not imply termination access. Exclude protected apps
	// before showing the closure review, without attempting to terminate them.
	terminationHandle, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return p, nil
	}
	windows.CloseHandle(terminationHandle)
	p.Safe = false
	return p, nil
}

// Terminate rechecks identity and safety on the same handle used to terminate.
// A recycled PID must never authorize killing an unrelated process.
func Terminate(expected Process) error {
	if expected.Safe {
		return fmt.Errorf("%w: %s", ErrProcessProtected, expected.Name)
	}
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, expected.PID)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	var c, e, k, u windows.Filetime
	if err = windows.GetProcessTimes(h, &c, &e, &k, &u); err != nil {
		return err
	}
	if c.Nanoseconds() != expected.Created {
		return errors.New("process identity changed")
	}
	p, err := Inspect(expected.PID, expected.Parent)
	if err != nil {
		return err
	}
	if p.Safe || !strings.EqualFold(p.Path, expected.Path) {
		return fmt.Errorf("%w: %s", ErrProcessProtected, expected.Name)
	}
	return windows.TerminateProcess(h, 0)
}
