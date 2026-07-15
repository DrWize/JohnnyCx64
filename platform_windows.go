//go:build windows

package main

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const (
	mbOK        = 0x00000000
	mbIconError = 0x00000010
	mbIconInfo  = 0x00000040
)

var messageBoxW = windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")

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
