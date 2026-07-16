//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

func normalizeWindowsScreenSaverArgs(args []string) ([]string, uintptr, bool, error) {
	return args, 0, false, nil
}

func attachPreviewWindow(_ unsafe.Pointer, parent uintptr) error {
	return fmt.Errorf("Windows screensaver preview handle %#x is unsupported on this platform", parent)
}

func previewHostAvailable(uintptr) bool {
	return false
}

func showErrorDialog(title, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, message)
}

func showInfoDialog(title, message string) {
	fmt.Fprintf(os.Stderr, "%s: %s\n", title, message)
}

func acquireSingleInstance() (func(), bool, error) {
	dir, err := appDataDir()
	if err != nil {
		return func() {}, false, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return func() {}, false, err
	}

	lockPath := filepath.Join(dir, "JohnnyCastaway.lock")
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return func() {}, false, err
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
			return func() {}, false, nil
		}
		return func() {}, false, err
	}

	release := func() {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
	}
	return release, true, nil
}
