package main

import (
	"fmt"
	"image/color"
	"log"
	"sort"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type menuEntry struct {
	label  string
	target string
}

var (
	menuVisible       bool
	menuEntries       []menuEntry
	menuSelection     int
	currentContent    string
	menuStatusText    string
	menuStatusUntil   float64
	menuSceneMessage  string
	crtFilterMode     crtFilter
	imageScalingMode  imageScaling
	fastCRTSharpness  = fastCRTBalanced
	menuFooterStarted float64
	escapeQuitArmed   bool
	escapeQuitAt      float64
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
	switch crtFilterMode {
	case crtOff:
		crtFilterMode = crtLightweight
	case crtLightweight:
		crtFilterMode = crtFast
	case crtFast:
		crtFilterMode = crtLottes
	default:
		crtFilterMode = crtOff
	}
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
	menuRevealFooter()
	if menuVisible || appSettings.windowed {
		rl.ShowCursor()
	}
}

func menuRevealFooter() {
	menuFooterStarted = rl.GetTime()
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

func screensaverFooterKeyPressed() bool {
	keys := []int32{
		rl.KeyF1, rl.KeyF2, rl.KeyF3, rl.KeyF4, rl.KeyF5, rl.KeyF7, rl.KeyF8, rl.KeyF9,
		rl.KeyN, rl.KeyT, rl.KeyH, rl.KeyEscape,
	}
	for _, key := range keys {
		if rl.IsKeyPressed(key) {
			return true
		}
	}
	return false
}

func menuDrawScreensaverFooter() {
	if !appSettings.screenSaver || menuVisible || traceVisible {
		return
	}
	opacity := menuFooterOpacity(rl.GetTime() - menuFooterStarted)
	if opacity <= 0 {
		return
	}

	width := int32(rl.GetScreenWidth())
	height := int32(rl.GetScreenHeight())
	footerHeight := int32(84)
	y := height - footerHeight
	background := rl.NewColor(12, 16, 24, uint8(220*opacity))
	border := rl.NewColor(145, 165, 195, uint8(210*opacity))
	foreground := rl.NewColor(235, 240, 248, uint8(255*opacity))
	accent := rl.NewColor(255, 203, 80, uint8(255*opacity))
	rl.DrawRectangle(0, y, width, footerHeight, background)
	rl.DrawRectangle(0, y, width, 1, border)

	lines := []struct {
		text  string
		color color.RGBA
	}{
		{fmt.Sprintf("F1 Settings | F2 CRT: %s | F3 Order: %s | F4 Scale: %s | F5 Log | F7 Fast: %s | F8 Stats | F9 Test",
			crtFilterMode.label(), storyPlaybackModeLabel(), imageScalingMode.label(), fastCRTSharpnessLabel(fastCRTSharpness)), accent},
		{"Up/Down Select  |  Enter Run  |  N Next TTM  |  T Next scene  |  H Holiday  |  Esc twice: Quit", foreground},
		{performanceFooterText() + "  |  Other input exits", foreground},
	}
	for index, line := range lines {
		fontSize := int32(15)
		for fontSize > 10 && rl.MeasureText(line.text, fontSize) > width-24 {
			fontSize--
		}
		textWidth := rl.MeasureText(line.text, fontSize)
		textX := (width - textWidth) / 2
		textY := y + 9 + int32(index)*24
		rl.DrawText(line.text, textX, textY, fontSize, line.color)
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
	menuShowStatus("Now running: " + menuEntries[menuSelection].label)
	menuRevealFooter()
}

func menuSetVisible(visible bool) {
	menuVisible = visible
	if visible || appSettings.windowed {
		rl.ShowCursor()
	} else {
		rl.HideCursor()
	}
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
	textWidth := rl.MeasureText(label, fontSize)
	rl.DrawText(label, int32(rect.X)+(int32(rect.Width)-textWidth)/2, int32(rect.Y)+(int32(rect.Height)-fontSize)/2, fontSize, rl.RayWhite)
	return hovered && rl.IsMouseButtonPressed(rl.MouseButtonLeft)
}

func menuUpdateAndDraw() {
	if appSettings.screenSaver && screensaverFooterKeyPressed() {
		menuRevealFooter()
	}
	menuDrawStatus()
	menuDrawScreensaverFooter()
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
	rl.DrawText("F1/Esc: hide  F2: CRT  F3: scene order  F4: scaling  F5: log", int32(panelX+24), int32(panelY+58), 16, rl.LightGray)
	rl.DrawText("Up/Down: choose  Enter: run  N: next TTM  T: scene  H: holiday  F7: sharp  F8: stats  F9: test", int32(panelX+24), int32(panelY+80), 14, rl.LightGray)
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
	toggleW := (panelW - 48 - toggleGap*2) / 3
	filterButton := rl.NewRectangle(panelX+24, toggleY, toggleW, 44)
	orderButton := rl.NewRectangle(filterButton.X+toggleW+toggleGap, toggleY, toggleW, 44)
	scalingButton := rl.NewRectangle(orderButton.X+toggleW+toggleGap, toggleY, toggleW, 44)
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

	buttonY := panelY + panelH - 134
	buttonGap := float32(12)
	buttonW := (panelW - 48 - buttonGap*2) / 3
	previous := rl.NewRectangle(panelX+24, buttonY, buttonW, 48)
	next := rl.NewRectangle(previous.X+buttonW+buttonGap, buttonY, buttonW, 48)
	run := rl.NewRectangle(next.X+buttonW+buttonGap, buttonY, buttonW, 48)

	shortcutY := panelY + panelH - 70
	shortcutW := (panelW - 48 - buttonGap*2) / 3
	nextTTM := rl.NewRectangle(panelX+24, shortcutY, shortcutW, 48)
	nextScene := rl.NewRectangle(nextTTM.X+shortcutW+buttonGap, shortcutY, shortcutW, 48)
	traceButton := rl.NewRectangle(nextScene.X+shortcutW+buttonGap, shortcutY, shortcutW, 48)

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
	if menuButton(traceButton, "Runtime log (F5)") {
		traceSetVisible(true)
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
