//go:build windows

package applications

import (
	"fmt"
	"unsafe"

	"github.com/Karench0/focus-lock/internal/model"
	"golang.org/x/sys/windows"
)

var assocQueryString = windows.NewLazySystemDLL("shlwapi.dll").NewProc("AssocQueryStringW")

// DefaultBrowser reads the current user's HTTPS handler without changing any
// association. Storozhenko mode requires a browser supported by our extension.
func DefaultBrowser() (model.Application, error) {
	if err := assocQueryString.Find(); err != nil {
		return model.Application{}, fmt.Errorf("Не удалось определить браузер по умолчанию / Could not detect the default browser: %w", err)
	}
	protocol, _ := windows.UTF16PtrFromString("https")
	verb, _ := windows.UTF16PtrFromString("open")
	buffer := make([]uint16, 32768)
	length := uint32(len(buffer))
	// ASSOCF_IS_PROTOCOL maps the protocol through the current user defaults;
	// ASSOCF_NOFIXUPS keeps this query read-only. ASSOCSTR_EXECUTABLE returns
	// the handler executable rather than a command.
	result, _, _ := assocQueryString.Call(
		0x00001000|0x00000100,
		2,
		uintptr(unsafe.Pointer(protocol)),
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(&length)),
	)
	if result != 0 || length == 0 || length > uint32(len(buffer)) {
		return model.Application{}, fmt.Errorf("Не удалось определить браузер по умолчанию / Could not detect the default browser (HRESULT 0x%08x)", uint32(result))
	}
	path, err := Canonical(windows.UTF16ToString(buffer[:length]))
	if err != nil {
		return model.Application{}, fmt.Errorf("Недоступен браузер по умолчанию / Default browser is unavailable: %w", err)
	}
	var name string
	switch Browser(path) {
	case "chrome":
		name = "Google Chrome"
	case "edge":
		name = "Microsoft Edge"
	default:
		return model.Application{}, fmt.Errorf("Выберите Chrome или Edge с расширением Focus Lock браузером по умолчанию / Set Chrome or Edge with the Focus Lock extension as your default browser")
	}
	return model.Application{Name: name, Path: path}, nil
}
