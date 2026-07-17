package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var screenshotRequested bool

func screenshotFilename(now time.Time) string {
	return "Johnny-Castaway-" + now.Format("20060102-150405.000") + ".png"
}

func requestScreenshot() {
	screenshotRequested = true
}

func captureScreenshot() (string, error) {
	directory, err := picturesDirectory()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create screenshot folder: %w", err)
	}
	path := filepath.Join(directory, screenshotFilename(time.Now()))
	rl.TakeScreenshot(path)
	if info, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("save screenshot: %w", err)
	} else if info.Size() == 0 {
		return "", fmt.Errorf("save screenshot: output file is empty")
	}
	return path, nil
}
