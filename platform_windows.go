//go:build windows

package main

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	mbOK        = 0x00000000
	mbIconError = 0x00000010
	mbIconInfo  = 0x00000040
)

var messageBoxW = windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")

var (
	setParentW      = windows.NewLazySystemDLL("user32.dll").NewProc("SetParent")
	getParentW      = windows.NewLazySystemDLL("user32.dll").NewProc("GetParent")
	getClientRectW  = windows.NewLazySystemDLL("user32.dll").NewProc("GetClientRect")
	moveWindowW     = windows.NewLazySystemDLL("user32.dll").NewProc("MoveWindow")
	isWindowW       = windows.NewLazySystemDLL("user32.dll").NewProc("IsWindow")
	setWindowLongW  = windows.NewLazySystemDLL("user32.dll").NewProc("SetWindowLongW")
	setWindowLongPW = windows.NewLazySystemDLL("user32.dll").NewProc("SetWindowLongPtrW")
)

const (
	gwlStyle  = -16
	wsChild   = 0x40000000
	wsVisible = 0x10000000
)

func normalizeWindowsScreenSaverArgs(args []string) ([]string, uintptr, bool, error) {
	if len(args) == 0 || !strings.HasPrefix(args[0], "/") {
		return args, 0, false, nil
	}

	mode := strings.ToLower(args[0])
	rest := args[1:]
	switch {
	case mode == "/s":
		return append([]string{"--screensaver"}, rest...), 0, false, nil
	case mode == "/p" || strings.HasPrefix(mode, "/p:"):
		value := strings.TrimPrefix(mode, "/p:")
		if mode == "/p" {
			if len(rest) == 0 {
				return nil, 0, false, fmt.Errorf("/p requires a preview parent window handle")
			}
			value, rest = rest[0], rest[1:]
		}
		parent, err := strconv.ParseUint(value, 0, 64)
		if err != nil || parent == 0 || uint64(uintptr(parent)) != parent {
			return nil, 0, false, fmt.Errorf("invalid /p preview parent window handle %q", value)
		}
		return append([]string{"--screensaver"}, rest...), uintptr(parent), false, nil
	case mode == "/c" || strings.HasPrefix(mode, "/c:"):
		return append([]string{"--windowed", "--menu"}, rest...), 0, true, nil
	default:
		return args, 0, false, nil
	}
}

func attachPreviewWindow(window unsafe.Pointer, parent uintptr) error {
	if window == nil || !previewHostAvailable(parent) {
		return fmt.Errorf("parent window %#x is unavailable", parent)
	}
	hwnd := uintptr(window)
	style := uintptr(wsChild | wsVisible)
	styleIndex := int32(gwlStyle)
	if strconv.IntSize == 64 {
		setWindowLongPW.Call(hwnd, uintptr(styleIndex), style)
	} else {
		setWindowLongW.Call(hwnd, uintptr(styleIndex), style)
	}
	setParentW.Call(hwnd, parent)
	if actualParent, _, callErr := getParentW.Call(hwnd); actualParent != parent {
		return fmt.Errorf("SetParent assigned %#x instead of %#x: %v", actualParent, parent, callErr)
	}
	var rect windows.Rect
	if result, _, callErr := getClientRectW.Call(parent, uintptr(unsafe.Pointer(&rect))); result == 0 {
		return fmt.Errorf("GetClientRect failed: %v", callErr)
	}
	width, height := rect.Right-rect.Left, rect.Bottom-rect.Top
	if width < 1 || height < 1 {
		return fmt.Errorf("parent window has an empty client area")
	}
	if result, _, callErr := moveWindowW.Call(hwnd, 0, 0, uintptr(width), uintptr(height), 1); result == 0 {
		return fmt.Errorf("MoveWindow failed: %v", callErr)
	}
	return nil
}

func previewHostAvailable(parent uintptr) bool {
	if parent == 0 {
		return false
	}
	result, _, _ := isWindowW.Call(parent)
	return result != 0
}

func showMessage(title, message string, flags uintptr) {
	titlePtr, titleErr := windows.UTF16PtrFromString(title)
	messagePtr, messageErr := windows.UTF16PtrFromString(message)
	if titleErr != nil || messageErr != nil {
		return
	}
	_, _, _ = messageBoxW.Call(0, uintptr(unsafePointer(messagePtr)), uintptr(unsafePointer(titlePtr)), flags)
}

func showErrorDialog(title, message string) {
	showMessage(title, message+"\n\n"+appVersionLabel(), mbOK|mbIconError)
}

func showInfoDialog(title, message string) {
	showMessage(title, message, mbOK|mbIconInfo)
}

func acquireSingleInstance() (func(), bool, error) {
	name, err := windows.UTF16PtrFromString("Local\\JohnnyCastaway2026")
	if err != nil {
		return func() {}, false, fmt.Errorf("mutex name: %w", err)
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if handle != 0 {
			_ = windows.CloseHandle(handle)
		}
		return func() {}, false, nil
	}
	if err != nil {
		return func() {}, false, err
	}
	return func() { _ = windows.CloseHandle(handle) }, true, nil
}
