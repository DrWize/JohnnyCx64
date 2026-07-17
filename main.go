package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	fadeInVal       = float32(255.0)
	appSettings     appOptions
	audioReady      bool
	resourcesLoaded bool
)

type appOptions struct {
	windowed       bool
	screenSaver    bool
	mute           bool
	stretch        bool
	monitor        int
	ttm            string
	menu           bool
	noSaveSettings bool
	crt            string
	dataDir        string
	previewParent  uintptr
	configuration  bool
}

type exitRequest struct{}

type contentSwitch struct {
	target string
}

func requestExit() {
	panic(exitRequest{})
}

func requestContentSwitch(target string) {
	panic(contentSwitch{target: target})
}

func parseOptions(args []string) (appOptions, error) {
	args, previewParent, configuration, err := normalizeWindowsScreenSaverArgs(args)
	if err != nil {
		return appOptions{monitor: 1}, err
	}
	opts := appOptions{monitor: 1}
	opts.previewParent = previewParent
	opts.configuration = configuration
	var fullscreen, soundEnabled, fit bool
	fs := flag.NewFlagSet("JohnnyCastaway", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&opts.windowed, "windowed", false, "run in a resizable window")
	fs.BoolVar(&fullscreen, "fullscreen", false, "use borderless fullscreen")
	fs.BoolVar(&opts.screenSaver, "screensaver", false, "screensaver input behavior with interactive F-key controls")
	fs.BoolVar(&opts.mute, "mute", false, "disable audio")
	fs.BoolVar(&soundEnabled, "sound", false, "enable audio")
	fs.BoolVar(&opts.stretch, "stretch", false, "stretch 4:3 graphics to fill the display")
	fs.BoolVar(&fit, "fit", false, "preserve the original 4:3 aspect ratio")
	fs.IntVar(&opts.monitor, "monitor", 1, "1-based monitor number")
	fs.StringVar(&opts.ttm, "ttm", "", "play one TTM resource directly")
	fs.BoolVar(&opts.menu, "menu", false, "open the hidden menu at startup")
	fs.BoolVar(&opts.noSaveSettings, "no-save-settings", false, "do not persist settings from this session")
	fs.StringVar(&opts.crt, "crt", "", "CRT mode: off, lightweight, fast, or lottes")
	fs.StringVar(&opts.dataDir, "data-dir", "", "folder containing RESOURCE.MAP, RESOURCE.001, and optional sound*.wav files")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if opts.windowed && fullscreen {
		return opts, fmt.Errorf("windowed and fullscreen cannot be used together")
	}
	if opts.mute && soundEnabled {
		return opts, fmt.Errorf("mute and sound cannot be used together")
	}
	if opts.stretch && fit {
		return opts, fmt.Errorf("stretch and fit cannot be used together")
	}
	if fullscreen {
		opts.windowed = false
	}
	if soundEnabled {
		opts.mute = false
	}
	if fit {
		opts.stretch = false
	}
	if opts.monitor < 1 {
		return opts, fmt.Errorf("monitor must be 1 or greater")
	}
	if opts.ttm != "" {
		opts.ttm = strings.ToUpper(opts.ttm)
		if !strings.HasSuffix(opts.ttm, ".TTM") {
			return opts, fmt.Errorf("ttm must name a .TTM resource")
		}
	}
	if opts.crt != "" {
		opts.crt = strings.ToLower(opts.crt)
		if mode := crtFilter(opts.crt); mode != crtOff && mode != crtLightweight && mode != crtFast && mode != crtLottes {
			return opts, fmt.Errorf("crt must be off, lightweight, fast, or lottes")
		}
	}
	return opts, nil
}

func main() {
	os.Exit(runApp())
}

func runApp() (exitCode int) {
	closeLog, err := initAppLogging()
	if err == nil {
		defer closeLog()
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			if _, ok := recovered.(exitRequest); ok {
				exitCode = 0
				return
			}
			message := fmt.Sprintf("%v", recovered)
			log.Printf("fatal error: %s\n%s", message, debug.Stack())
			showErrorDialog("Johnny Castaway", "Johnny Castaway could not continue.\n\n"+message+"\n\nSee the log file in LocalAppData for details.")
			exitCode = 1
		}
	}()

	rawArgs := os.Args[1:]
	opts, err := parseOptions(rawArgs)
	if err != nil {
		showErrorDialog("Johnny Castaway", "Invalid command line: "+err.Error())
		return 2
	}
	appSettings = opts
	normalizedArgs, _, _, _ := normalizeWindowsScreenSaverArgs(rawArgs)
	loadPersistentSettings(normalizedArgs)
	if err := loadResourceArchivesForStartup(); err != nil {
		showErrorDialog("Johnny Castaway data files", err.Error()+"\n\nRun the application or /c configuration mode to browse for your original files, or use --data-dir.")
		return 3
	}

	releaseInstance := func() {}
	if appSettings.previewParent == 0 {
		var firstInstance bool
		releaseInstance, firstInstance, err = acquireSingleInstance()
		if err != nil {
			panic(fmt.Errorf("single-instance check: %w", err))
		}
		if !firstInstance {
			showInfoDialog("Johnny Castaway", "Johnny Castaway is already running.")
			return 0
		}
	}
	defer releaseInstance()
	if !appSettings.noSaveSettings {
		persistDisplayPlaybackSettings()
	}

	log.Printf("starting: windowed=%t screensaver=%t previewParent=%#x configuration=%t mute=%t stretch=%t monitor=%d ttm=%q menu=%t dataDir=%q", appSettings.windowed, appSettings.screenSaver, appSettings.previewParent, appSettings.configuration, appSettings.mute, appSettings.stretch, appSettings.monitor, appSettings.ttm, appSettings.menu, appSettings.dataDir)
	currentContent = opts.ttm
	runEngine(func() { runContentSessions(opts.ttm) })
	return 0
}

func loadResourceArchivesForStartup() error {
	err := loadResourceArchives(appSettings.dataDir)
	if err == nil || (appSettings.screenSaver && !appSettings.configuration) {
		return err
	}

	selected, accepted, pickerErr := chooseDataDirectory(0)
	if pickerErr != nil {
		return fmt.Errorf("select data folder: %w", pickerErr)
	}
	if !accepted {
		return err
	}
	if err := loadResourceArchives(selected); err != nil {
		return fmt.Errorf("selected data folder is invalid: %w", err)
	}
	appSettings.dataDir = selected
	persistDataDirectory(selected)
	return nil
}

func runContentSessions(target string) {
	for {
		nextTarget, switched := runContentSession(target)
		if !switched {
			return
		}
		log.Printf("switching content in existing window from %q to %q", target, nextTarget)
		releaseContentSession()
		target = nextTarget
	}
}

func runContentSession(target string) (nextTarget string, switched bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if change, ok := recovered.(contentSwitch); ok {
				nextTarget = change.target
				switched = true
				return
			}
			panic(recovered)
		}
	}()

	currentContent = target
	menuContentChanged(target)
	if target == "" {
		storyPlay()
	} else {
		for {
			adsPlaySingleTtm(target)
		}
	}
	return "", false
}

func releaseContentSession() {
	soundStopAll()
	graphicsReleaseContent()
}

func setupApp() {
	isPreview := appSettings.previewParent != 0
	toggleableWindow := !appSettings.screenSaver && !isPreview
	flags := uint32(rl.FlagMsaa4xHint | rl.FlagWindowHighdpi)
	if appSettings.windowed || toggleableWindow {
		flags |= rl.FlagWindowResizable
	} else {
		flags |= rl.FlagWindowUndecorated
	}
	if appSettings.screenSaver && !isPreview {
		flags |= rl.FlagWindowTopmost
	}
	rl.SetConfigFlags(flags)
	rl.InitWindow(screenWidth, screenHeight, "Johnny Castaway")
	// Escape is handled by the application's double-press quit policy.
	rl.SetExitKey(rl.KeyNull)

	if isPreview {
		if err := attachPreviewWindow(rl.GetWindowHandle(), appSettings.previewParent); err != nil {
			panic(fmt.Errorf("screensaver preview: %w", err))
		}
		log.Printf("display: embedded screensaver preview in parent %#x", appSettings.previewParent)
	} else {
		monitorCount := rl.GetMonitorCount()
		mon := resolveMonitorIndex(appSettings.monitor, monitorCount)
		if mon != appSettings.monitor-1 {
			log.Printf("requested monitor %d is unavailable; using monitor 1", appSettings.monitor)
		}
		monW := rl.GetMonitorWidth(mon)
		monH := rl.GetMonitorHeight(mon)
		if monW <= 0 {
			monW = 1920
		}
		if monH <= 0 {
			monH = 1080
		}
		monPos := rl.GetMonitorPosition(mon)
		log.Printf("display: monitor=%d origin=%dx%d size=%dx%d aspect=%.3f ultrawide=%t",
			mon+1, int(monPos.X), int(monPos.Y), monW, monH,
			float64(monW)/float64(monH), is32By9Display(monW, monH))

		winW, winH := defaultWindowedSize(monW, monH)
		if toggleableWindow {
			// Start from the centered decorated size even when fullscreen was
			// requested. Raylib then remembers these bounds and can restore them
			// when F toggles borderless fullscreen off.
			rl.SetWindowSize(winW, winH)
			rl.SetWindowPosition(int(monPos.X)+(monW-winW)/2, int(monPos.Y)+(monH-winH)/2)
			if !appSettings.windowed {
				rl.ToggleBorderlessWindowed()
			}
		} else if appSettings.windowed {
			rl.SetWindowSize(winW, winH)
			rl.SetWindowPosition(int(monPos.X)+(monW-winW)/2, int(monPos.Y)+(monH-winH)/2)
		} else {
			rl.SetWindowSize(monW, monH)
			rl.SetWindowPosition(int(monPos.X), int(monPos.Y))
		}
		if !appSettings.windowed {
			rl.DisableCursor()
			rl.HideCursor()
		}
	}

	if !appSettings.mute {
		rl.InitAudioDevice()
		audioReady = rl.IsAudioDeviceReady()
		if audioReady {
			rl.SetMasterVolume(1.0)
			if err := loadSfx(); err != nil {
				log.Printf("audio warning: %v", err)
			}
		} else {
			log.Printf("audio device is unavailable; continuing without sound")
		}
	}

	rl.SetTargetFPS(30)

	if !resourcesLoaded {
		parseResourceFiles(filepath.Join(appSettings.dataDir, "RESOURCE.MAP"))
		resourcesLoaded = true
	}
	menuInitialize(currentContent)

	doFadeIn()
	// The playback loop already schedules frames at 30 FPS. Disable Raylib's
	// additional limiter after the startup fade so playback is not throttled
	// twice and render-cost measurements exclude an artificial wait.
	rl.SetTargetFPS(0)
	graphicsInit()
}

func defaultWindowedSize(monitorWidth, monitorHeight int) (width, height int) {
	width, height = 960, 720
	if width > monitorWidth {
		width = monitorWidth
	}
	if height > monitorHeight {
		height = monitorHeight
	}
	return width, height
}

func resolveMonitorIndex(requested, monitorCount int) int {
	index := requested - 1
	if monitorCount < 1 || index < 0 || index >= monitorCount {
		return 0
	}
	return index
}

func is32By9Display(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	// Allow for bezel-compensated and custom resolutions near the 32:9 ratio.
	return float64(width)/float64(height) >= 3.4
}

func doFadeIn() {
	fadeInVal = 255.0

	for !rl.WindowShouldClose() {
		if appSettings.previewParent != 0 && !previewHostAvailable(appSettings.previewParent) {
			requestExit()
		}
		rl.BeginDrawing()
		rl.ClearBackground(rl.Blank)

		alpha := 1.0 - fadeInVal/255.0
		rl.DrawRectangle(0, 0, int32(rl.GetScreenWidth()), int32(rl.GetScreenHeight()), rl.Fade(rl.Black, alpha))
		fadeInVal -= 10
		rl.EndDrawing()

		if fadeInVal <= 0 {
			return
		}
	}
	requestExit()
}

func runEngine(play func()) {
	audioReady = false
	setupApp()
	defer rl.CloseWindow()
	defer graphicsEnd()
	if audioReady {
		defer rl.CloseAudioDevice()
		defer unloadSfx()
	}

	play()
}
