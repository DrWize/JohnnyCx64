package main

import (
	"fmt"
	"log"
	"sort"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type menuEntry struct {
	label  string
	target string
}

type shortcutDockItem struct {
	key    string
	action string
}

var (
	menuVisible           bool
	menuEntries           []menuEntry
	menuSelection         int
	currentContent        string
	menuStatusText        string
	menuStatusUntil       float64
	menuSceneMessage      string
	crtFilterMode         crtFilter
	imageScalingMode      imageScaling
	fastCRTSharpness      = fastCRTBalanced
	menuFooterStarted     float64
	uiLastActivity        float64
	uiPreviousMouse       rl.Vector2
	uiMouseInitialized    bool
	escapeQuitArmed       bool
	escapeQuitAt          float64
	dataManagerVisible    bool
	dataManagerMessage    string
	dataManagerValid      bool
	dayNightStatusPending string
)

const (
	menuFooterDuration = 10.0
	menuFooterFadeTime = 2.0
	doubleEscapeWindow = 1.5
)

func menuShowStatus(text string) {
	menuStatusText = text
	menuStatusUntil = rl.GetTime() + 4
}

func cycleCRTFilter() {
	performanceBenchmarkCancel()
	crtFilterMode = detectedCRTCapabilities.next(crtFilterMode)
	performanceModeChanged()
	persistDisplayPlaybackSettings()
}

func cycleFastCRTSharpness() {
	performanceBenchmarkCancel()
	fastCRTSharpness = (normalizeFastCRTSharpness(fastCRTSharpness) + 1) % 3
	applyFastCRTSharpness()
	performanceModeChanged()
	persistDisplayPlaybackSettings()
}

func menuInitialize(current string) {
	menuVisible = appSettings.menu
	menuEntries = []menuEntry{{label: "Full story", target: ""}}

	names := make([]string, 0, numTtmResources)
	for i := 0; i < numTtmResources; i++ {
		names = append(names, ttmResources[i].ResName)
	}
	sort.Strings(names)
	for _, name := range names {
		menuEntries = append(menuEntries, menuEntry{label: name, target: name})
	}

	menuSelection = menuIndexForTarget(current)
	menuShowStatus("Now running: " + menuEntries[menuSelection].label)
	dataManagerRefresh()
	menuRevealFooter()
	uiInitializeActivity(rl.GetTime(), rl.GetMousePosition())
	if menuVisible || appSettings.windowed {
		rl.ShowCursor()
	}
}

func uiInitializeActivity(now float64, mouse rl.Vector2) {
	uiLastActivity = now
	uiPreviousMouse = mouse
	uiMouseInitialized = true
}

func uiObserveActivity(now float64, mouse rl.Vector2, mouseButtonPressed, keyPressed bool) bool {
	mouseMoved := uiMouseInitialized && mouse != uiPreviousMouse
	uiPreviousMouse = mouse
	uiMouseInitialized = true
	if !mouseMoved && !mouseButtonPressed && !keyPressed {
		return false
	}
	uiLastActivity = now
	menuFooterStarted = now
	return true
}

func informationalUIOpacity(now float64) float32 {
	return menuFooterOpacity(now - uiLastActivity)
}

func menuRevealFooter() {
	now := rl.GetTime()
	menuFooterStarted = now
	uiLastActivity = now
}

func menuFooterOpacity(elapsed float64) float32 {
	if elapsed < 0 {
		return 1
	}
	remaining := menuFooterDuration - elapsed
	if remaining <= 0 {
		return 0
	}
	if remaining < menuFooterFadeTime {
		return float32(remaining / menuFooterFadeTime)
	}
	return 1
}

func shortcutDockItems(screenSaver bool) []shortcutDockItem {
	items := []shortcutDockItem{
		{key: "F1", action: "Settings"},
		{key: "F2", action: "CRT " + crtFilterMode.label()},
		{key: "F3", action: "Order " + storyPlaybackModeLabel()},
		{key: "F4", action: "Scale " + imageScalingMode.label()},
		{key: "F5", action: "Runtime log"},
		{key: "F7", action: "Sharp " + fastCRTSharpnessLabel(fastCRTSharpness)},
		{key: "F8", action: "Stats"},
		{key: "F9", action: "Benchmark"},
		{key: "F10", action: "Data files"},
	}
	if !screenSaver {
		items = append(items, shortcutDockItem{key: "F", action: "Fullscreen"})
	}
	return append(items,
		shortcutDockItem{key: "D", action: "Day"},
		shortcutDockItem{key: "N", action: "Next TTM"},
		shortcutDockItem{key: "T", action: "Next scene"},
		shortcutDockItem{key: "H", action: "Holiday"},
		shortcutDockItem{key: "↑ ↓", action: "Select"},
		shortcutDockItem{key: "Enter", action: "Run"},
		shortcutDockItem{key: "Esc ×2", action: "Quit"},
	)
}

func menuDrawShortcutDock() {
	if foregroundOverlayVisible() {
		return
	}
	opacity := informationalUIOpacity(rl.GetTime())
	if opacity <= 0 {
		return
	}

	fontSize := int32(13)
	chipHeight := float32(26)
	gap := float32(7)
	panelMargin := float32(16)
	panelPadding := float32(12)
	contentWidth := float32(rl.GetScreenWidth()) - panelMargin*2 - panelPadding*2
	items := shortcutDockItems(appSettings.screenSaver)
	type placedShortcut struct {
		item shortcutDockItem
		rect rl.Rectangle
	}
	placed := make([]placedShortcut, 0, len(items))
	x := float32(0)
	y := float32(0)
	for _, item := range items {
		label := item.key + "  " + item.action
		chipWidth := float32(rl.MeasureText(label, fontSize)) + 22
		if x > 0 && x+chipWidth > contentWidth {
			x = 0
			y += chipHeight + gap
		}
		placed = append(placed, placedShortcut{item: item, rect: rl.NewRectangle(x, y, chipWidth, chipHeight)})
		x += chipWidth + gap
	}

	rowsHeight := y + chipHeight
	headerHeight := float32(30)
	panelHeight := panelPadding*2 + headerHeight + rowsHeight
	panel := rl.NewRectangle(panelMargin, float32(rl.GetScreenHeight())-panelHeight-panelMargin, float32(rl.GetScreenWidth())-panelMargin*2, panelHeight)
	background := rl.Fade(rl.NewColor(10, 14, 22, 242), opacity)
	border := rl.Fade(rl.NewColor(116, 145, 184, 210), opacity)
	foreground := rl.Fade(rl.NewColor(226, 233, 243, 255), opacity)
	muted := rl.Fade(rl.NewColor(158, 171, 190, 255), opacity)
	accent := rl.Fade(rl.NewColor(255, 197, 72, 255), opacity)
	rl.DrawRectangleRounded(panel, 0.14, 10, background)
	rl.DrawRectangleRoundedLinesEx(panel, 0.14, 10, 1, border)

	titleY := int32(panel.Y + panelPadding + 1)
	rl.DrawText("SHORTCUTS", int32(panel.X+panelPadding), titleY, 14, accent)
	status := performanceFooterText()
	if appSettings.screenSaver {
		status += "  •  Other input exits"
	}
	statusSize := int32(12)
	for statusSize > 9 && rl.MeasureText(status, statusSize) > int32(panel.Width-panelPadding*2)-110 {
		statusSize--
	}
	statusWidth := rl.MeasureText(status, statusSize)
	rl.DrawText(status, int32(panel.X+panel.Width-panelPadding)-statusWidth, titleY+1, statusSize, muted)

	for _, shortcut := range placed {
		rect := shortcut.rect
		rect.X += panel.X + panelPadding
		rect.Y += panel.Y + panelPadding + headerHeight
		rl.DrawRectangleRounded(rect, 0.35, 8, rl.Fade(rl.NewColor(35, 43, 57, 238), opacity))
		rl.DrawRectangleRoundedLinesEx(rect, 0.35, 8, 1, rl.Fade(rl.NewColor(75, 91, 115, 220), opacity))
		textY := int32(rect.Y + (rect.Height-float32(fontSize))/2)
		keyX := int32(rect.X + 11)
		rl.DrawText(shortcut.item.key, keyX, textY, fontSize, accent)
		actionX := keyX + rl.MeasureText(shortcut.item.key, fontSize) + 9
		rl.DrawText(shortcut.item.action, actionX, textY, fontSize, foreground)
	}
}

func doubleEscapeShouldExit(armed bool, previous, now float64) bool {
	return armed && now >= previous && now-previous <= doubleEscapeWindow
}

func handleDoubleEscape() {
	if !rl.IsKeyPressed(rl.KeyEscape) {
		return
	}
	now := rl.GetTime()
	if doubleEscapeShouldExit(escapeQuitArmed, escapeQuitAt, now) {
		requestExit()
	}
	escapeQuitArmed = true
	escapeQuitAt = now
	menuShowStatus("Press Esc again within 1.5 seconds to quit")
	menuRevealFooter()
}

func menuIndexForTarget(target string) int {
	for i, entry := range menuEntries {
		if entry.target == target {
			return i
		}
	}
	return 0
}

func menuContentChanged(target string) {
	menuSelection = menuIndexForTarget(target)
	menuSceneMessage = ""
	if dayNightStatusPending != "" {
		menuSceneMessage = dayNightStatusPending
		menuShowStatus(dayNightStatusPending)
		dayNightStatusPending = ""
	} else {
		menuShowStatus("Now running: " + menuEntries[menuSelection].label)
	}
	menuRevealFooter()
}

func menuSetVisible(visible bool) {
	menuVisible = visible
	if visible || dataManagerVisible || traceVisible || appSettings.windowed {
		rl.ShowCursor()
	} else {
		rl.HideCursor()
	}
}

func foregroundOverlayVisible() bool {
	return menuVisible || traceVisible || dataManagerVisible
}

func dataManagerSetVisible(visible bool) {
	dataManagerVisible = visible
	if visible {
		menuVisible = false
		traceSetVisible(false)
		dataManagerRefresh()
		rl.ShowCursor()
	} else if !menuVisible && !traceVisible && !appSettings.windowed {
		rl.HideCursor()
	}
}

func dataManagerRefresh() {
	if err := validateDataDirectory(appSettings.dataDir); err != nil {
		dataManagerValid = false
		dataManagerMessage = err.Error()
		return
	}
	dataManagerValid = true
	dataManagerMessage = "Verified canonical RESOURCE.MAP and RESOURCE.001"
}

func compactMiddle(value string, limit int) string {
	runes := []rune(value)
	if limit < 5 || len(runes) <= limit {
		return value
	}
	left := (limit - 3) / 2
	right := limit - 3 - left
	return string(runes[:left]) + "..." + string(runes[len(runes)-right:])
}

func dataManagerChooseFolder() {
	selected, accepted, err := chooseDataDirectory(uintptr(rl.GetWindowHandle()))
	if err != nil {
		dataManagerValid = false
		dataManagerMessage = "Folder picker failed: " + err.Error()
		return
	}
	if !accepted {
		dataManagerMessage = "Folder selection canceled; current setting was kept."
		return
	}
	if err := validateDataDirectory(selected); err != nil {
		dataManagerValid = validateDataDirectory(appSettings.dataDir) == nil
		dataManagerMessage = "Not saved: " + err.Error()
		return
	}
	appSettings.dataDir = selected
	persistDataDirectory(selected)
	dataManagerValid = true
	dataManagerMessage = "Verified and saved. Optional sound changes apply after restart."
	menuShowStatus("Data folder saved: " + compactMiddle(selected, 48))
}

func dataManagerOpenFolder() {
	if err := openDirectory(uintptr(rl.GetWindowHandle()), appSettings.dataDir); err != nil {
		dataManagerMessage = err.Error()
		return
	}
	dataManagerMessage = "Opened the current data folder."
}

func menuMoveSelection(delta int) {
	if len(menuEntries) == 0 {
		return
	}
	menuSelection = (menuSelection + delta + len(menuEntries)) % len(menuEntries)
}

func menuRunSelected() {
	if len(menuEntries) == 0 {
		return
	}
	requestContentSwitch(menuEntries[menuSelection].target)
}

func menuRunNextTTM() {
	if len(menuEntries) < 2 {
		return
	}

	current := menuIndexForTarget(currentContent)
	next := current + 1
	if next >= len(menuEntries) {
		next = 1 // Skip "Full story" when wrapping through TTMs.
	}
	menuSelection = next
	requestContentSwitch(menuEntries[next].target)
}

func menuRunNextScene() {
	if currentContent == "" {
		menuSceneMessage = "Next scene is available while a TTM is running."
		menuShowStatus(menuSceneMessage)
		return
	}

	slot := &ttmSlots[0]
	nextTag, ok := nextTTMSceneIndex(slot, ttmThreads[0].ip)
	if !ok {
		menuSceneMessage = "This TTM has no selectable scenes."
		menuShowStatus(menuSceneMessage)
		return
	}

	ttmThreads[0].ip = slot.tags[nextTag].offset
	ttmThreads[0].timer = 0
	ttmThreads[0].delay = 0
	ttmThreads[0].nextGotoOffset = 0
	ttmThreads[0].isRunning = 1
	grUpdateDelay = 0

	description := ttmSceneDescription(nextTag)
	menuSceneMessage = fmt.Sprintf("Scene %d: %s", nextTag+1, description)
	menuShowStatus(fmt.Sprintf("Now running: %s — %s", currentContent, menuSceneMessage))
	menuSetVisible(false)
	log.Printf("advanced %s to scene %d (%s)", currentContent, nextTag+1, description)
}

func menuCycleDayNightPreview() {
	menuApplyDayNightPreview(islandCycleDayNight())
}

func menuSetDayPreview() {
	menuApplyDayNightPreview(islandSetDayNightOverride(0))
}

func menuApplyDayNightPreview(label string) {
	menuSceneMessage = "Day/night preview: " + label
	if currentContent == "" {
		// Restart the Full Story content session so the background changes at
		// once, but skip its introductory title screen for this preview action.
		dayNightStatusPending = menuSceneMessage
		storySkipIntroOnce = true
		requestContentSwitch(currentContent)
	}
	menuShowStatus(menuSceneMessage + " (used by Full Story)")
}

func validTTMSceneIndexes(slot *TTtmSlot) []int {
	if slot == nil {
		return nil
	}
	indexes := make([]int, 0, slot.numTags)
	for i := 0; i < slot.numTags; i++ {
		if slot.tags[i].id != 0 && slot.tags[i].offset < slot.dataSize {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func nextTTMSceneIndex(slot *TTtmSlot, instructionPointer uint32) (int, bool) {
	indexes := validTTMSceneIndexes(slot)
	if len(indexes) == 0 {
		return 0, false
	}

	currentPosition := -1
	for position, index := range indexes {
		if slot.tags[index].offset <= instructionPointer {
			currentPosition = position
		}
	}
	return indexes[(currentPosition+1)%len(indexes)], true
}

func currentTTMSceneInfo() (number, total int, description string, ok bool) {
	if currentContent == "" {
		return 0, 0, "", false
	}
	slot := &ttmSlots[0]
	indexes := validTTMSceneIndexes(slot)
	if len(indexes) == 0 {
		return 0, 0, "", false
	}

	position := -1
	for i, index := range indexes {
		if slot.tags[index].offset <= ttmThreads[0].ip {
			position = i
		}
	}
	if position < 0 {
		return 0, len(indexes), "Initialization", true
	}
	index := indexes[position]
	return position + 1, len(indexes), ttmSceneDescription(index), true
}

func ttmSceneDescription(tagIndex int) string {
	for i := 0; i < numTtmResources; i++ {
		resource := &ttmResources[i]
		if resource.ResName == currentContent && tagIndex < len(resource.Tags) {
			description := resource.Tags[tagIndex].Description
			if description != "" {
				return description
			}
		}
	}
	return fmt.Sprintf("Tag %d", tagIndex+1)
}

func menuButton(rect rl.Rectangle, label string) bool {
	hovered := rl.CheckCollisionPointRec(rl.GetMousePosition(), rect)
	background := rl.NewColor(48, 54, 66, 255)
	if hovered {
		background = rl.NewColor(72, 92, 120, 255)
	}
	rl.DrawRectangleRec(rect, background)
	rl.DrawRectangleLinesEx(rect, 1, rl.NewColor(145, 165, 195, 255))
	fontSize := int32(20)
	for fontSize > 12 && rl.MeasureText(label, fontSize) > int32(rect.Width)-12 {
		fontSize--
	}
	textWidth := rl.MeasureText(label, fontSize)
	rl.DrawText(label, int32(rect.X)+(int32(rect.Width)-textWidth)/2, int32(rect.Y)+(int32(rect.Height)-fontSize)/2, fontSize, rl.RayWhite)
	return hovered && rl.IsMouseButtonPressed(rl.MouseButtonLeft)
}

func menuUpdateAndDraw() {
	menuDrawStatus()
	menuDrawShortcutDock()
	controlDown := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	if rl.IsKeyPressed(rl.KeyF) && !controlDown && !traceVisible && !appSettings.screenSaver && appSettings.previewParent == 0 {
		grToggleFullscreen()
		mode := "Fullscreen"
		if appSettings.windowed {
			mode = "Windowed"
		}
		menuShowStatus("Window mode: " + mode)
	}
	if rl.IsKeyPressed(rl.KeyF10) {
		dataManagerSetVisible(!dataManagerVisible)
	}
	if dataManagerVisible {
		dataManagerUpdateAndDraw()
		return
	}
	if rl.IsKeyPressed(rl.KeyF2) {
		cycleCRTFilter()
		menuShowStatus("CRT filter: " + crtFilterMode.label())
	}
	if rl.IsKeyPressed(rl.KeyF3) {
		storyTogglePlaybackMode()
		menuShowStatus("Scene order: " + storyPlaybackModeLabel())
	}
	if rl.IsKeyPressed(rl.KeyF4) {
		cycleImageScalingMode()
		menuShowStatus("Image scaling: " + imageScalingMode.label())
	}
	if rl.IsKeyPressed(rl.KeyF7) {
		cycleFastCRTSharpness()
		menuShowStatus("Fast CRT sharpness: " + fastCRTSharpnessLabel(fastCRTSharpness))
	}
	if rl.IsKeyPressed(rl.KeyF8) {
		performanceToggle()
		menuShowStatus("Performance display: " + onOffLabel(performancePinned))
	}
	if rl.IsKeyPressed(rl.KeyF9) {
		performanceBenchmarkToggle()
		if performanceBenchmarkActive {
			menuShowStatus("Shader benchmark started")
		} else {
			menuShowStatus("Shader benchmark stopped")
		}
	}
	if rl.IsKeyPressed(rl.KeyN) {
		menuRunNextTTM()
	}
	if rl.IsKeyPressed(rl.KeyD) && !traceVisible {
		menuSetDayPreview()
	}
	if rl.IsKeyPressed(rl.KeyT) {
		menuRunNextScene()
	}
	if rl.IsKeyPressed(rl.KeyH) {
		label := islandCycleHoliday()
		menuSceneMessage = "Holiday preview: " + label
		menuShowStatus(menuSceneMessage)
	}

	if rl.IsKeyPressed(rl.KeyF1) {
		menuSetVisible(!menuVisible)
	}
	if menuVisible && rl.IsKeyPressed(rl.KeyEscape) {
		menuSetVisible(false)
		return
	}
	if !menuVisible {
		return
	}

	if rl.IsKeyPressed(rl.KeyUp) {
		menuMoveSelection(-1)
	}
	if rl.IsKeyPressed(rl.KeyDown) {
		menuMoveSelection(1)
	}
	if rl.IsKeyPressed(rl.KeyEnter) {
		menuRunSelected()
	}
	windowW := float32(rl.GetScreenWidth())
	windowH := float32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, int32(windowW), int32(windowH), rl.Fade(rl.Black, 0.65))

	panelW := min(float32(620), windowW-40)
	panelH := min(float32(500), windowH-40)
	panelX := (windowW - panelW) / 2
	panelY := (windowH - panelH) / 2
	panel := rl.NewRectangle(panelX, panelY, panelW, panelH)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && !rl.CheckCollisionPointRec(rl.GetMousePosition(), panel) {
		menuSetVisible(false)
		return
	}
	rl.DrawRectangleRec(panel, rl.NewColor(24, 28, 36, 248))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(145, 165, 195, 255))

	rl.DrawText("Johnny Castaway settings", int32(panelX+24), int32(panelY+20), 28, rl.RayWhite)
	buildLabel := appVersionLabel()
	buildWidth := rl.MeasureText(buildLabel, 15)
	rl.DrawText(buildLabel, int32(panelX+panelW-24)-buildWidth, int32(panelY+26), 15, rl.Gray)
	rl.DrawText("F1/Esc: hide  F: fullscreen  F2: CRT  F3: order  F4: scaling  F5: log  F10: data", int32(panelX+24), int32(panelY+58), 14, rl.LightGray)
	rl.DrawText("Up/Down: choose  Enter: run  D: day  N: next TTM  T: scene  H: holiday  F7: sharp  F8: stats  F9: test", int32(panelX+24), int32(panelY+80), 14, rl.LightGray)
	if appSettings.screenSaver {
		rl.DrawText("Screensaver continues behind this panel; unlisted input exits.", int32(panelX+24), int32(panelY+102), 15, rl.Gold)
	}

	currentLabel := menuEntries[menuIndexForTarget(currentContent)].label
	selectedLabel := menuEntries[menuSelection].label
	contentOffset := float32(0)
	if appSettings.screenSaver {
		contentOffset = 18
	}
	rl.DrawText("Running: "+currentLabel, int32(panelX+24), int32(panelY+112+contentOffset), 19, rl.NewColor(150, 205, 255, 255))
	if number, total, description, ok := currentTTMSceneInfo(); ok {
		sceneText := fmt.Sprintf("Scene: %d/%d — %s", number, total, description)
		rl.DrawText(sceneText, int32(panelX+24), int32(panelY+138+contentOffset), 17, rl.NewColor(175, 210, 180, 255))
	}
	rl.DrawText(fmt.Sprintf("Choose TTM (%d/%d):", menuSelection+1, len(menuEntries)), int32(panelX+24), int32(panelY+170+contentOffset), 18, rl.LightGray)
	rl.DrawText(selectedLabel, int32(panelX+24), int32(panelY+196+contentOffset), 25, rl.Gold)
	if menuSceneMessage != "" {
		rl.DrawText(menuSceneMessage, int32(panelX+24), int32(panelY+230+contentOffset), 16, rl.LightGray)
	}
	guideY := panelY + panelH - 236
	rl.DrawText("Filter impact guide (F8 live timing, F9 comparison test)", int32(panelX+24), int32(guideY), 15, rl.LightGray)
	rl.DrawText("Off: Minimal   Lightweight: Very low   Fast: Low   Lottes: High", int32(panelX+24), int32(guideY+20), 15, rl.Gold)

	toggleY := panelY + panelH - 198
	toggleGap := float32(12)
	toggleW := (panelW - 48 - toggleGap*3) / 4
	filterButton := rl.NewRectangle(panelX+24, toggleY, toggleW, 44)
	orderButton := rl.NewRectangle(filterButton.X+toggleW+toggleGap, toggleY, toggleW, 44)
	scalingButton := rl.NewRectangle(orderButton.X+toggleW+toggleGap, toggleY, toggleW, 44)
	dayNightButton := rl.NewRectangle(scalingButton.X+toggleW+toggleGap, toggleY, toggleW, 44)
	if menuButton(filterButton, "CRT: "+crtFilterMode.label()) {
		cycleCRTFilter()
		menuShowStatus("CRT filter: " + crtFilterMode.label())
	}
	if menuButton(orderButton, "Scenes: "+storyPlaybackModeLabel()) {
		storyTogglePlaybackMode()
		menuShowStatus("Scene order: " + storyPlaybackModeLabel())
	}
	if menuButton(scalingButton, "Scale: "+imageScalingMode.label()) {
		cycleImageScalingMode()
		menuShowStatus("Image scaling: " + imageScalingMode.label())
	}
	if menuButton(dayNightButton, "Sky: "+islandDayNightLabel()) {
		menuCycleDayNightPreview()
	}

	buttonY := panelY + panelH - 134
	buttonGap := float32(12)
	buttonW := (panelW - 48 - buttonGap*2) / 3
	previous := rl.NewRectangle(panelX+24, buttonY, buttonW, 48)
	next := rl.NewRectangle(previous.X+buttonW+buttonGap, buttonY, buttonW, 48)
	run := rl.NewRectangle(next.X+buttonW+buttonGap, buttonY, buttonW, 48)

	shortcutY := panelY + panelH - 70
	shortcutW := (panelW - 48 - buttonGap*3) / 4
	nextTTM := rl.NewRectangle(panelX+24, shortcutY, shortcutW, 48)
	nextScene := rl.NewRectangle(nextTTM.X+shortcutW+buttonGap, shortcutY, shortcutW, 48)
	dataButton := rl.NewRectangle(nextScene.X+shortcutW+buttonGap, shortcutY, shortcutW, 48)
	traceButton := rl.NewRectangle(dataButton.X+shortcutW+buttonGap, shortcutY, shortcutW, 48)

	if menuButton(previous, "Prev") {
		menuMoveSelection(-1)
	}
	if menuButton(next, "Next") {
		menuMoveSelection(1)
	}
	if menuButton(run, "Run") {
		menuRunSelected()
	}
	if menuButton(nextTTM, "Next TTM (N)") {
		menuRunNextTTM()
	}
	if menuButton(nextScene, "Next scene (T)") {
		menuRunNextScene()
	}
	if menuButton(dataButton, "Data files (F10)") {
		dataManagerSetVisible(true)
	}
	if menuButton(traceButton, "Runtime log (F5)") {
		traceSetVisible(true)
	}
}

func dataManagerUpdateAndDraw() {
	if rl.IsKeyPressed(rl.KeyEscape) {
		dataManagerSetVisible(false)
		return
	}
	if rl.IsKeyPressed(rl.KeyF1) {
		dataManagerSetVisible(false)
		menuSetVisible(true)
		return
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyB) {
		dataManagerChooseFolder()
	}
	if rl.IsKeyPressed(rl.KeyR) {
		dataManagerRefresh()
	}
	if rl.IsKeyPressed(rl.KeyO) {
		dataManagerOpenFolder()
	}

	windowW := float32(rl.GetScreenWidth())
	windowH := float32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, int32(windowW), int32(windowH), rl.Fade(rl.Black, 0.72))
	panelW := min(float32(740), windowW-40)
	panelH := min(float32(360), windowH-40)
	panelX := (windowW - panelW) / 2
	panelY := (windowH - panelH) / 2
	panel := rl.NewRectangle(panelX, panelY, panelW, panelH)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && !rl.CheckCollisionPointRec(rl.GetMousePosition(), panel) {
		dataManagerSetVisible(false)
		return
	}
	rl.DrawRectangleRec(panel, rl.NewColor(22, 27, 36, 252))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(112, 170, 225, 255))
	rl.DrawText("Data files", int32(panelX+26), int32(panelY+22), 30, rl.RayWhite)
	rl.DrawText("F10/Esc close   Enter/B browse   R recheck   O open folder   F1 settings", int32(panelX+26), int32(panelY+62), 16, rl.LightGray)

	card := rl.NewRectangle(panelX+26, panelY+96, panelW-52, 142)
	rl.DrawRectangleRec(card, rl.NewColor(34, 41, 54, 255))
	rl.DrawRectangleLinesEx(card, 1, rl.NewColor(92, 112, 140, 255))
	rl.DrawText("Current folder", int32(card.X+18), int32(card.Y+16), 17, rl.LightGray)
	pathText := compactMiddle(appSettings.dataDir, 78)
	rl.DrawText(pathText, int32(card.X+18), int32(card.Y+44), 19, rl.RayWhite)
	statusColor := rl.NewColor(235, 120, 110, 255)
	statusLabel := "Needs attention"
	if dataManagerValid {
		statusColor = rl.NewColor(115, 220, 145, 255)
		statusLabel = "Ready"
	}
	rl.DrawText(statusLabel, int32(card.X+18), int32(card.Y+80), 20, statusColor)
	rl.DrawText(compactMiddle(dataManagerMessage, 88), int32(card.X+18), int32(card.Y+108), 15, rl.LightGray)

	buttonGap := float32(14)
	buttonY := panelY + panelH - 82
	buttonW := (panelW - 52 - buttonGap*2) / 3
	browse := rl.NewRectangle(panelX+26, buttonY, buttonW, 50)
	recheck := rl.NewRectangle(browse.X+buttonW+buttonGap, buttonY, buttonW, 50)
	open := rl.NewRectangle(recheck.X+buttonW+buttonGap, buttonY, buttonW, 50)
	if menuButton(browse, "Browse... (Enter/B)") {
		dataManagerChooseFolder()
	}
	if menuButton(recheck, "Recheck (R)") {
		dataManagerRefresh()
	}
	if menuButton(open, "Open folder (O)") {
		dataManagerOpenFolder()
	}
}

func onOffLabel(enabled bool) string {
	if enabled {
		return "On"
	}
	return "Off"
}

func menuDrawStatus() {
	if menuVisible || menuStatusText == "" || rl.GetTime() >= menuStatusUntil {
		return
	}

	fontSize := int32(20)
	padding := int32(14)
	textWidth := rl.MeasureText(menuStatusText, fontSize)
	boxWidth := textWidth + padding*2
	boxHeight := fontSize + padding*2
	x := int32(rl.GetScreenWidth()) - boxWidth - 18
	y := int32(18)

	rl.DrawRectangle(x, y, boxWidth, boxHeight, rl.NewColor(24, 28, 36, 230))
	rl.DrawRectangleLines(x, y, boxWidth, boxHeight, rl.NewColor(145, 165, 195, 255))
	rl.DrawText(menuStatusText, x+padding, y+padding, fontSize, rl.RayWhite)
}
