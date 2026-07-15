package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const traceHistoryLimit = 2000

type traceSource uint8

const (
	traceSourceTTM traceSource = iota
	traceSourceADS
)

type traceView uint8

const (
	traceViewHistory traceView = iota
	traceViewADSScript
)

type traceFilter uint8

const (
	traceFilterAll traceFilter = iota
	traceFilterTTM
	traceFilterADS
)

type ttmTraceEntry struct {
	sequence    uint64
	source      traceSource
	script      string
	ttm         string
	sceneID     uint16
	sceneNumber int
	sceneTotal  int
	sceneName   string
	offset      uint32
	opcode      uint16
	instruction string
}

type adsTraceLine struct {
	offset      uint32
	opcode      uint16
	instruction string
}

var (
	traceVisible         bool
	traceHistory         []ttmTraceEntry
	traceSequence        uint64
	traceScroll          int
	tracePaused          bool
	traceCurrentView     = traceViewHistory
	traceCurrentFilter   = traceFilterAll
	traceSearch          string
	traceSearchActive    bool
	traceStatus          string
	traceStatusUntil     float64
	traceADSName         string
	traceADSLines        []adsTraceLine
	traceADSActiveOffset uint32
	traceADSHasActive    bool
)

var traceOpcodeNames = map[uint16]string{
	0x0080: "DRAW_BACKGROUND",
	0x0110: "PURGE",
	0x0FF0: "UPDATE",
	0x1021: "SET_DELAY",
	0x1051: "SET_BMP_SLOT",
	0x1061: "SET_PALETTE_SLOT",
	0x1101: "LOCAL_SCENE",
	0x1111: "SET_SCENE",
	0x1121: "DEFINE_REGION",
	0x1201: "GOTO_SCENE",
	0x2002: "SET_COLORS",
	0x2012: "SET_FRAME",
	0x2022: "TIMER",
	0x4004: "SET_CLIP_ZONE",
	0x4204: "COPY_ZONE_TO_BG",
	0x4214: "SAVE_IMAGE",
	0xA002: "DRAW_PIXEL",
	0xA054: "SAVE_ZONE",
	0xA064: "RESTORE_ZONE",
	0xA0A4: "DRAW_LINE",
	0xA104: "DRAW_RECT",
	0xA404: "DRAW_CIRCLE",
	0xA504: "DRAW_SPRITE",
	0xA524: "DRAW_SPRITE_FLIP",
	0xA601: "CLEAR_SCREEN",
	0xB606: "DRAW_SCREEN",
	0xC051: "PLAY_SAMPLE",
	0xF01F: "LOAD_SCREEN",
	0xF02F: "LOAD_IMAGE",
	0xF05F: "LOAD_PALETTE",
}

var adsOpcodeNames = map[uint16]string{
	0x1070: "WHILE_RUNNING",
	0x1330: "IF_NOT_PLAYED",
	0x1350: "IF_FINISHED",
	0x1360: "IF_NOT_RUNNING",
	0x1370: "IF_IS_RUNNING",
	0x1420: "AND",
	0x1430: "OR",
	0x1510: "END_IF",
	0x1520: "END_WHILE",
	0x2005: "ADD_SCENE",
	0x2010: "STOP_SCENE",
	0x2014: "UNKNOWN_2014",
	0x3010: "RANDOM_START",
	0x3020: "RANDOM_NOP",
	0x30FF: "RANDOM_END",
	0x4000: "UNKNOWN_4000",
	0xF010: "FADE_OUT",
	0xF200: "GOSUB_TAG",
	0xFFF0: "END_IF",
	0xFFFF: "END",
}

var adsOpcodeArgCounts = map[uint16]int{
	0x1070: 2,
	0x1330: 2,
	0x1350: 2,
	0x1360: 2,
	0x1370: 2,
	0x1420: 0,
	0x1430: 0,
	0x1510: 0,
	0x1520: 0,
	0x2005: 4,
	0x2010: 3,
	0x2014: 0,
	0x3010: 0,
	0x3020: 1,
	0x30FF: 0,
	0x4000: 3,
	0xF010: 0,
	0xF200: 1,
	0xFFF0: 0,
	0xFFFF: 0,
}

func traceSetVisible(visible bool) {
	traceVisible = visible
	if visible {
		traceScroll = 0
		rl.ShowCursor()
	} else if !menuVisible && !appSettings.windowed {
		rl.HideCursor()
	}
}

func traceToggle() {
	traceSetVisible(!traceVisible)
}

func traceRecordTTMInstruction(thread *TTtmThread, offset uint32, opcode uint16, args []uint16, stringArg string) {
	if tracePaused || thread == nil || thread.ttmSlot == nil {
		return
	}

	slot := thread.ttmSlot
	sceneID, sceneNumber, sceneName := traceSceneAt(slot, offset)
	if opcode == 0x1101 || opcode == 0x1111 {
		if len(args) > 0 {
			sceneID = args[0]
			for i, tag := range slot.tags {
				if tag.id == sceneID {
					sceneNumber = i + 1
					sceneName = traceSceneDescription(slot.name, i)
					break
				}
			}
		}
	}

	traceAppend(ttmTraceEntry{
		source:      traceSourceTTM,
		script:      slot.name,
		ttm:         slot.name,
		sceneID:     sceneID,
		sceneNumber: sceneNumber,
		sceneTotal:  len(validTTMSceneIndexes(slot)),
		sceneName:   sceneName,
		offset:      offset,
		opcode:      opcode,
		instruction: traceFormatInstruction(opcode, args, stringArg),
	})
}

func traceBeginADS(name string, data []byte) {
	if tracePaused {
		return
	}
	traceADSName = name
	traceADSLines = decodeADSScript(data)
	traceADSHasActive = false
}

func traceRecordADSInstruction(offset uint32) {
	if tracePaused {
		return
	}
	line, ok := traceADSLineAt(offset)
	if !ok {
		return
	}
	traceADSActiveOffset = offset
	traceADSHasActive = true
	traceAppend(ttmTraceEntry{
		source:      traceSourceADS,
		script:      traceADSName,
		offset:      offset,
		opcode:      line.opcode,
		instruction: line.instruction,
	})
}

func traceAppend(entry ttmTraceEntry) {
	traceSequence++
	entry.sequence = traceSequence
	traceHistory = append(traceHistory, entry)
	if len(traceHistory) > traceHistoryLimit {
		traceHistory = traceHistory[len(traceHistory)-traceHistoryLimit:]
	}
}

func decodeADSScript(data []byte) []adsTraceLine {
	lines := make([]adsTraceLine, 0, len(data)/4)
	for offset := 0; offset+2 <= len(data); {
		instructionOffset := offset
		opcode := binary.LittleEndian.Uint16(data[offset:])
		offset += 2
		argCount, known := adsOpcodeArgCounts[opcode]
		if !known {
			lines = append(lines, adsTraceLine{
				offset:      uint32(instructionOffset),
				opcode:      opcode,
				instruction: fmt.Sprintf("TAG %d", opcode),
			})
			continue
		}
		if offset+argCount*2 > len(data) {
			lines = append(lines, adsTraceLine{
				offset:      uint32(instructionOffset),
				opcode:      opcode,
				instruction: fmt.Sprintf("%s <truncated>", adsOpcodeName(opcode)),
			})
			break
		}
		args := make([]uint16, argCount)
		for i := range args {
			args[i] = binary.LittleEndian.Uint16(data[offset:])
			offset += 2
		}
		lines = append(lines, adsTraceLine{
			offset:      uint32(instructionOffset),
			opcode:      opcode,
			instruction: traceFormatNamedInstruction(adsOpcodeName(opcode), args),
		})
	}
	return lines
}

func adsOpcodeName(opcode uint16) string {
	if name, ok := adsOpcodeNames[opcode]; ok {
		return name
	}
	return fmt.Sprintf("ADS_%04X", opcode)
}

func traceADSLineAt(offset uint32) (adsTraceLine, bool) {
	for _, line := range traceADSLines {
		if line.offset == offset {
			return line, true
		}
	}
	return adsTraceLine{}, false
}

func traceSceneAt(slot *TTtmSlot, offset uint32) (id uint16, number int, description string) {
	if slot == nil {
		return 0, 0, "Initialization"
	}
	for i, tag := range slot.tags {
		if tag.id != 0 && tag.offset <= offset {
			id = tag.id
			number = i + 1
			description = traceSceneDescription(slot.name, i)
		}
	}
	if number == 0 {
		description = "Initialization"
	}
	return id, number, description
}

func traceSceneDescription(ttmName string, tagIndex int) string {
	for i := 0; i < numTtmResources; i++ {
		resource := &ttmResources[i]
		if resource.ResName == ttmName && tagIndex >= 0 && tagIndex < len(resource.Tags) {
			if resource.Tags[tagIndex].Description != "" {
				return resource.Tags[tagIndex].Description
			}
		}
	}
	if tagIndex >= 0 {
		return fmt.Sprintf("Scene %d", tagIndex+1)
	}
	return "Initialization"
}

func traceFormatInstruction(opcode uint16, args []uint16, stringArg string) string {
	name := traceOpcodeName(opcode)
	if stringArg != "" {
		return fmt.Sprintf("%s %q", name, stringArg)
	}
	return traceFormatNamedInstruction(name, args)
}

func traceFormatNamedInstruction(name string, args []uint16) string {
	if len(args) == 0 {
		return name
	}
	values := make([]string, len(args))
	for i, arg := range args {
		values[i] = fmt.Sprintf("%d", arg)
	}
	return name + " " + strings.Join(values, " ")
}

func traceOpcodeName(opcode uint16) string {
	if name, ok := traceOpcodeNames[opcode]; ok {
		return name
	}
	return fmt.Sprintf("OPCODE_%04X", opcode)
}

func traceHistoryEntries() []ttmTraceEntry {
	query := strings.ToLower(strings.TrimSpace(traceSearch))
	entries := make([]ttmTraceEntry, 0, len(traceHistory))
	for _, entry := range traceHistory {
		if traceCurrentFilter == traceFilterTTM && entry.source != traceSourceTTM {
			continue
		}
		if traceCurrentFilter == traceFilterADS && entry.source != traceSourceADS {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(traceEntryText(entry)), query) {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func traceADSScriptLines() []adsTraceLine {
	query := strings.ToLower(strings.TrimSpace(traceSearch))
	if query == "" {
		return append([]adsTraceLine(nil), traceADSLines...)
	}
	lines := make([]adsTraceLine, 0, len(traceADSLines))
	for _, line := range traceADSLines {
		text := fmt.Sprintf("%04X %04X %s", line.offset, line.opcode, line.instruction)
		if strings.Contains(strings.ToLower(text), query) {
			lines = append(lines, line)
		}
	}
	return lines
}

func traceEntryText(entry ttmTraceEntry) string {
	source := "TTM"
	if entry.source == traceSourceADS {
		source = "ADS"
	}
	scene := ""
	if entry.source == traceSourceTTM {
		scene = fmt.Sprintf(" scene=%d", entry.sceneID)
	}
	return fmt.Sprintf("%06d %s %-15s %04X %04X%s %s", entry.sequence, source, entry.script, entry.offset, entry.opcode, scene, entry.instruction)
}

func traceCurrentViewText() string {
	var builder strings.Builder
	if traceCurrentView == traceViewADSScript {
		fmt.Fprintf(&builder, "ADS script: %s\n", traceADSName)
		for _, line := range traceADSScriptLines() {
			marker := " "
			if traceADSHasActive && line.offset == traceADSActiveOffset {
				marker = ">"
			}
			fmt.Fprintf(&builder, "%s %04X  %04X  %s\n", marker, line.offset, line.opcode, line.instruction)
		}
		return builder.String()
	}
	fmt.Fprintf(&builder, "Runtime history (filter: %s, search: %q)\n", traceFilterLabel(), traceSearch)
	for _, entry := range traceHistoryEntries() {
		builder.WriteString(traceEntryText(entry))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func traceCopyCurrentView() {
	rl.SetClipboardText(traceCurrentViewText())
	traceShowStatus("Copied current view to clipboard")
}

func traceExportCurrentView() {
	dir, err := appDataDir()
	if err != nil {
		traceShowStatus("Export failed: " + err.Error())
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		traceShowStatus("Export failed: " + err.Error())
		return
	}
	name := "JohnnyCastaway-trace-" + time.Now().Format("20060102-150405") + ".txt"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(traceCurrentViewText()), 0644); err != nil {
		traceShowStatus("Export failed: " + err.Error())
		return
	}
	traceShowStatus("Exported " + name)
}

func traceShowStatus(value string) {
	traceStatus = value
	traceStatusUntil = rl.GetTime() + 4
}

func traceCycleFilter() {
	traceCurrentFilter = (traceCurrentFilter + 1) % 3
	traceScroll = 0
	traceShowStatus("Filter: " + traceFilterLabel())
}

func traceFilterLabel() string {
	switch traceCurrentFilter {
	case traceFilterTTM:
		return "TTM"
	case traceFilterADS:
		return "ADS"
	default:
		return "All"
	}
}

func traceViewLabel() string {
	if traceCurrentView == traceViewADSScript {
		return "ADS script"
	}
	return "History"
}

func traceToggleView() {
	if traceCurrentView == traceViewHistory {
		traceCurrentView = traceViewADSScript
	} else {
		traceCurrentView = traceViewHistory
	}
	traceScroll = 0
}

func traceHandleInput() bool {
	control := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)
	if traceSearchActive {
		if rl.IsKeyPressed(rl.KeyEscape) || rl.IsKeyPressed(rl.KeyEnter) {
			traceSearchActive = false
			return true
		}
		if rl.IsKeyPressed(rl.KeyBackspace) && traceSearch != "" {
			_, size := utf8.DecodeLastRuneInString(traceSearch)
			traceSearch = traceSearch[:len(traceSearch)-size]
			traceScroll = 0
		}
		for char := rl.GetCharPressed(); char != 0; char = rl.GetCharPressed() {
			if char >= 32 && char < 127 && len(traceSearch) < 80 {
				traceSearch += string(char)
				traceScroll = 0
			}
		}
		return false
	}

	if control && rl.IsKeyPressed(rl.KeyF) {
		traceSearchActive = true
		return true
	}
	if control && rl.IsKeyPressed(rl.KeyC) {
		traceCopyCurrentView()
	}
	if control && rl.IsKeyPressed(rl.KeyE) {
		traceExportCurrentView()
	}
	if control && rl.IsKeyPressed(rl.KeyL) {
		traceHistory = nil
		traceScroll = 0
		traceShowStatus("Runtime history cleared")
	}
	if rl.IsKeyPressed(rl.KeySpace) {
		tracePaused = !tracePaused
		traceShowStatus("Capture: " + onOffLabel(!tracePaused))
	}
	if rl.IsKeyPressed(rl.KeyTab) {
		traceToggleView()
	}
	if rl.IsKeyPressed(rl.KeyF6) {
		traceCycleFilter()
	}
	return false
}

func traceUpdateAndDraw() {
	if rl.IsKeyPressed(rl.KeyF5) {
		traceToggle()
	}
	if !traceVisible {
		return
	}
	if traceHandleInput() {
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		traceSetVisible(false)
		return
	}

	if rl.IsKeyPressed(rl.KeyPageUp) {
		traceScroll += 12
	}
	if rl.IsKeyPressed(rl.KeyPageDown) {
		traceScroll -= 12
	}
	if rl.IsKeyPressed(rl.KeyEnd) {
		traceScroll = 0
	}
	wheel := rl.GetMouseWheelMove()
	if wheel > 0 {
		traceScroll += 3
	} else if wheel < 0 {
		traceScroll -= 3
	}

	windowW := float32(rl.GetScreenWidth())
	windowH := float32(rl.GetScreenHeight())
	panelW := min(float32(1080), windowW-30)
	panelH := min(float32(700), windowH-30)
	panelX := (windowW - panelW) / 2
	panelY := (windowH - panelH) / 2
	panel := rl.NewRectangle(panelX, panelY, panelW, panelH)

	rl.DrawRectangle(0, 0, int32(windowW), int32(windowH), rl.Fade(rl.Black, 0.76))
	rl.DrawRectangleRec(panel, rl.NewColor(15, 19, 25, 252))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(125, 190, 145, 255))
	rl.DrawText("Runtime diagnostics", int32(panelX+20), int32(panelY+14), 27, rl.RayWhite)
	rl.DrawText("F5/Esc close  Tab view  F6 filter  Ctrl+F search  Space pause  Ctrl+C copy  Ctrl+E export", int32(panelX+20), int32(panelY+46), 15, rl.LightGray)

	searchRect := rl.NewRectangle(panelX+20, panelY+70, panelW-40, 32)
	searchColor := rl.NewColor(30, 36, 44, 255)
	if traceSearchActive {
		searchColor = rl.NewColor(45, 58, 70, 255)
	}
	rl.DrawRectangleRec(searchRect, searchColor)
	rl.DrawRectangleLinesEx(searchRect, 1, rl.NewColor(110, 140, 160, 255))
	searchText := "Search: " + traceSearch
	if traceSearchActive {
		searchText += "_"
	} else if traceSearch == "" {
		searchText += "Ctrl+F or click here"
	}
	rl.DrawText(searchText, int32(searchRect.X+10), int32(searchRect.Y+7), 17, rl.RayWhite)
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) && rl.CheckCollisionPointRec(rl.GetMousePosition(), searchRect) {
		traceSearchActive = true
	}

	buttonY := panelY + 112
	gap := float32(8)
	buttonW := (panelW - 40 - gap*5) / 6
	buttons := []struct {
		label string
		run   func()
	}{
		{"View: " + traceViewLabel(), traceToggleView},
		{"Capture: " + onOffLabel(!tracePaused), func() { tracePaused = !tracePaused }},
		{"Filter: " + traceFilterLabel(), traceCycleFilter},
		{"Copy", traceCopyCurrentView},
		{"Export", traceExportCurrentView},
		{"Clear history", func() { traceHistory = nil; traceScroll = 0 }},
	}
	for i, button := range buttons {
		rect := rl.NewRectangle(panelX+20+float32(i)*(buttonW+gap), buttonY, buttonW, 38)
		if menuButton(rect, button.label) {
			button.run()
		}
	}

	if traceStatus != "" && rl.GetTime() < traceStatusUntil {
		rl.DrawText(traceStatus, int32(panelX+20), int32(panelY+158), 16, rl.Gold)
	}

	listTop := int32(panelY + 184)
	listBottom := int32(panelY + panelH - 20)
	if traceCurrentView == traceViewADSScript {
		traceDrawADSScript(panelX, panelW, listTop, listBottom)
	} else {
		traceDrawHistory(panelX, panelW, listTop, listBottom)
	}
}

func traceDrawHistory(panelX, panelW float32, listTop, listBottom int32) {
	entries := traceHistoryEntries()
	if len(entries) == 0 {
		rl.DrawText("No runtime instructions match the current filter/search.", int32(panelX+22), listTop+10, 19, rl.Gold)
		return
	}

	latest := entries[len(entries)-1]
	header := fmt.Sprintf("%s %s  offset 0x%04X  opcode 0x%04X  %s", traceSourceLabel(latest.source), latest.script, latest.offset, latest.opcode, latest.instruction)
	rl.DrawText(traceFitText(header, int32(panelW-44), 17), int32(panelX+22), listTop, 17, rl.NewColor(150, 205, 255, 255))
	listTop += 30

	lineHeight := int32(21)
	visibleLines := max(1, int((listBottom-listTop)/lineHeight))
	maxScroll := max(0, len(entries)-visibleLines)
	traceScroll = max(0, min(traceScroll, maxScroll))
	end := len(entries) - traceScroll
	start := max(0, end-visibleLines)
	currentSequence := traceHistory[len(traceHistory)-1].sequence

	for i := start; i < end; i++ {
		entry := entries[i]
		y := listTop + int32(i-start)*lineHeight
		isCurrent := entry.sequence == currentSequence && traceScroll == 0
		if isCurrent {
			rl.DrawRectangle(int32(panelX+16), y-2, int32(panelW-32), lineHeight, rl.NewColor(42, 72, 54, 230))
		}
		marker := " "
		if isCurrent {
			marker = ">"
		}
		line := marker + " " + traceEntryText(entry)
		color := rl.LightGray
		if entry.source == traceSourceADS {
			color = rl.NewColor(215, 190, 125, 255)
		}
		if isCurrent {
			color = rl.RayWhite
		}
		rl.DrawText(traceFitText(line, int32(panelW-48), 15), int32(panelX+24), y, 15, color)
	}
}

func traceDrawADSScript(panelX, panelW float32, listTop, listBottom int32) {
	if traceADSName == "" {
		rl.DrawText("No ADS script has executed yet. Run Full Story to populate this view.", int32(panelX+22), listTop+10, 19, rl.Gold)
		return
	}
	state := "last executed"
	if tracePaused {
		state = "paused at"
	}
	header := fmt.Sprintf("ADS: %s  %s 0x%04X  decoded instructions: %d", traceADSName, state, traceADSActiveOffset, len(traceADSLines))
	rl.DrawText(header, int32(panelX+22), listTop, 18, rl.NewColor(150, 205, 255, 255))
	listTop += 30

	lines := traceADSScriptLines()
	lineHeight := int32(21)
	visibleLines := max(1, int((listBottom-listTop)/lineHeight))
	maxScroll := max(0, len(lines)-visibleLines)
	traceScroll = max(0, min(traceScroll, maxScroll))
	end := len(lines) - traceScroll
	start := max(0, end-visibleLines)
	if traceScroll == 0 && traceADSHasActive {
		for i, line := range lines {
			if line.offset == traceADSActiveOffset {
				start = max(0, min(i-visibleLines/2, maxScroll))
				end = min(len(lines), start+visibleLines)
				break
			}
		}
	}

	for i := start; i < end; i++ {
		line := lines[i]
		y := listTop + int32(i-start)*lineHeight
		active := traceADSHasActive && line.offset == traceADSActiveOffset
		if active {
			rl.DrawRectangle(int32(panelX+16), y-2, int32(panelW-32), lineHeight, rl.NewColor(76, 61, 28, 235))
		}
		marker := " "
		color := rl.LightGray
		if active {
			marker = ">"
			color = rl.Gold
		}
		text := fmt.Sprintf("%s %04X  %04X  %s", marker, line.offset, line.opcode, line.instruction)
		rl.DrawText(traceFitText(text, int32(panelW-48), 16), int32(panelX+24), y, 16, color)
	}
}

func traceSourceLabel(source traceSource) string {
	if source == traceSourceADS {
		return "ADS"
	}
	return "TTM"
}

func traceFitText(value string, maxWidth, fontSize int32) string {
	if rl.MeasureText(value, fontSize) <= maxWidth {
		return value
	}
	for len(value) > 3 && rl.MeasureText(value+"...", fontSize) > maxWidth {
		value = value[:len(value)-1]
	}
	return value + "..."
}
