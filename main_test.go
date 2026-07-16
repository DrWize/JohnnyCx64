package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    appOptions
		wantErr bool
	}{
		{name: "defaults", want: appOptions{monitor: 1}},
		{name: "Windows screensaver", args: []string{"/s"}, want: appOptions{monitor: 1, screenSaver: true}},
		{name: "Windows preview separated handle", args: []string{"/p", "1234"}, want: appOptions{monitor: 1, screenSaver: true, previewParent: 1234}},
		{name: "Windows preview attached handle", args: []string{"/P:0x4d2"}, want: appOptions{monitor: 1, screenSaver: true, previewParent: 1234}},
		{name: "Windows configuration", args: []string{"/c:1234"}, want: appOptions{monitor: 1, windowed: true, menu: true, configuration: true}},
		{name: "missing preview handle", args: []string{"/p"}, wantErr: true},
		{name: "invalid preview handle", args: []string{"/p:not-a-window"}, wantErr: true},
		{name: "Fast CRT override", args: []string{"--crt", "FAST"}, want: appOptions{monitor: 1, crt: "fast"}},
		{
			name: "all options",
			args: []string{"--windowed", "--screensaver", "--mute", "--stretch", "--monitor", "2", "--ttm", "fire.ttm", "--menu", "--data-dir", "C:\\Johnny"},
			want: appOptions{windowed: true, screenSaver: true, mute: true, stretch: true, monitor: 2, ttm: "FIRE.TTM", menu: true, dataDir: "C:\\Johnny"},
		},
		{name: "invalid monitor", args: []string{"--monitor", "0"}, wantErr: true},
		{name: "invalid CRT mode", args: []string{"--crt", "royale"}, wantErr: true},
		{name: "conflicting window modes", args: []string{"--windowed", "--fullscreen"}, wantErr: true},
		{name: "conflicting audio modes", args: []string{"--mute", "--sound"}, wantErr: true},
		{name: "conflicting aspect modes", args: []string{"--stretch", "--fit"}, wantErr: true},
		{name: "invalid TTM extension", args: []string{"--ttm", "FIRE.BMP"}, wantErr: true},
		{name: "unknown option", args: []string{"--unknown"}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseOptions(test.args)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseOptions() error = %v, wantErr %t", err, test.wantErr)
			}
			if !test.wantErr && !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parseOptions() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestNextTTMSceneIndex(t *testing.T) {
	slot := &TTtmSlot{
		dataSize: 100,
		numTags:  5,
		tags: []TTtmTag{
			{id: 0, offset: 5},
			{id: 1, offset: 10},
			{id: 2, offset: 40},
			{id: 3, offset: 100},
			{id: 4, offset: 80},
		},
	}

	tests := []struct {
		name   string
		ip     uint32
		want   int
		wantOK bool
	}{
		{name: "before first scene", ip: 0, want: 1, wantOK: true},
		{name: "inside first scene", ip: 20, want: 2, wantOK: true},
		{name: "wrap after last scene", ip: 90, want: 1, wantOK: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := nextTTMSceneIndex(slot, test.ip)
			if got != test.want || ok != test.wantOK {
				t.Fatalf("nextTTMSceneIndex(%d) = (%d, %t), want (%d, %t)", test.ip, got, ok, test.want, test.wantOK)
			}
		})
	}

	if _, ok := nextTTMSceneIndex(&TTtmSlot{}, 0); ok {
		t.Fatal("nextTTMSceneIndex() reported a scene for an empty slot")
	}
}

func TestMenuRunNextTTMWraps(t *testing.T) {
	oldEntries, oldContent, oldSelection := menuEntries, currentContent, menuSelection
	t.Cleanup(func() {
		menuEntries, currentContent, menuSelection = oldEntries, oldContent, oldSelection
	})

	menuEntries = []menuEntry{
		{label: "Full story", target: ""},
		{label: "A.TTM", target: "A.TTM"},
		{label: "B.TTM", target: "B.TTM"},
	}
	currentContent = "B.TTM"

	defer func() {
		recovered := recover()
		change, ok := recovered.(contentSwitch)
		if !ok {
			t.Fatalf("menuRunNextTTM() panic = %#v, want contentSwitch", recovered)
		}
		if change.target != "A.TTM" {
			t.Fatalf("menuRunNextTTM() target = %q, want A.TTM", change.target)
		}
		if menuSelection != 1 {
			t.Fatalf("menuRunNextTTM() selection = %d, want 1", menuSelection)
		}
	}()

	menuRunNextTTM()
	t.Fatal("menuRunNextTTM() did not request a content switch")
}

func TestTTMLoadScansStringBearingOpcode(t *testing.T) {
	data := make([]byte, 0, 20)
	data = binary.LittleEndian.AppendUint16(data, 0xF02F)
	data = append(data, []byte("FLAME.BMP")...)
	data = append(data, 0)
	data = binary.LittleEndian.AppendUint16(data, 0x1111)
	data = binary.LittleEndian.AppendUint16(data, 7)
	data = binary.LittleEndian.AppendUint16(data, 0x0FF0)

	oldResources, oldCount := ttmResources, numTtmResources
	t.Cleanup(func() { ttmResources, numTtmResources = oldResources, oldCount })
	ttmResources = make([]TTTMResource, MaxTTMResources)
	ttmResources[0] = TTTMResource{
		ResName:          "STRING.TTM",
		UncompressedSize: uint32(len(data)),
		UncompressedData: data,
		NumTags:          1,
	}
	numTtmResources = 1

	var slot TTtmSlot
	ttmLoadTTM(&slot, "STRING.TTM")
	if slot.numTags != 1 || slot.tags[0].id != 7 || slot.tags[0].offset != 16 {
		t.Fatalf("ttmLoadTTM() tags = %#v, want tag 7 at offset 16", slot.tags)
	}
}

func TestMissingFireSpritesAreSkipped(t *testing.T) {
	oldResources, oldCount := bmpResources, numBmpResources
	t.Cleanup(func() { bmpResources, numBmpResources = oldResources, oldCount })
	bmpResources = make([]TBMPResource, MaxBMPResources)
	numBmpResources = 0

	for _, name := range []string{"FLAME.BMP", "FLURRY.BMP"} {
		t.Run(name, func(t *testing.T) {
			slot := &TTtmSlot{}
			grLoadBmp(slot, 0, name)
			if slot.numSprites[0] != 0 {
				t.Fatalf("grLoadBmp(%q) loaded %d sprites, want 0", name, slot.numSprites[0])
			}
		})
	}
}

func TestFireTTMMissingSpriteUsage(t *testing.T) {
	resetEmbeddedResourcesForTest(t)
	parseResourceFiles("assets/RESOURCE.MAP")
	resource := findTTMResource("FIRE.TTM")
	data := resource.UncompressedData
	selectedSlot := uint16(0)
	loaded := make(map[uint16]string)
	missingDraws := make(map[string]int)
	maxFrame := make(map[string]uint16)

	for offset := 0; offset+2 <= len(data); {
		opcode := binary.LittleEndian.Uint16(data[offset:])
		offset += 2
		argCount := int(opcode & 0x000f)
		if argCount == 0x0f {
			start := offset
			for offset < len(data) && data[offset] != 0 {
				offset++
			}
			value := string(data[start:offset])
			offset++
			if offset%2 != 0 {
				offset++
			}
			if opcode == 0xF02F {
				loaded[selectedSlot] = value
			}
			continue
		}
		if offset+argCount*2 > len(data) {
			t.Fatalf("FIRE.TTM has truncated opcode %#04x", opcode)
		}
		args := make([]uint16, argCount)
		for i := range args {
			args[i] = binary.LittleEndian.Uint16(data[offset+i*2:])
		}
		if opcode == 0x1051 {
			selectedSlot = args[0]
		}
		if opcode == 0xA504 || opcode == 0xA524 {
			name := loaded[args[3]]
			if name == "FLAME.BMP" || name == "FLURRY.BMP" {
				missingDraws[name]++
				if args[2] > maxFrame[name] {
					maxFrame[name] = args[2]
				}
				t.Logf("%s frame=%d x=%d y=%d flipped=%t", name, args[2], int16(args[0]), int16(args[1]), opcode == 0xA524)
			}
		}
		offset += argCount * 2
	}

	for _, name := range []string{"FLAME.BMP", "FLURRY.BMP"} {
		if missingDraws[name] == 0 {
			t.Errorf("FIRE.TTM never draws %s", name)
		}
		t.Logf("%s draws=%d maxFrame=%d", name, missingDraws[name], maxFrame[name])
	}
}

func TestDominantBottomColor(t *testing.T) {
	pixels := []byte{
		1, 1, 1, 255, 2, 2, 2, 255, 3, 3, 3, 255,
		9, 8, 7, 255, 4, 5, 6, 255, 9, 8, 7, 255,
	}
	if got, want := dominantBottomColor(pixels, 3, 2), (color.RGBA{R: 9, G: 8, B: 7, A: 255}); got != want {
		t.Fatalf("dominantBottomColor() = %#v, want %#v", got, want)
	}

	if got, want := dominantBottomColor(nil, 0, 0), (color.RGBA{A: 255}); got != want {
		t.Fatalf("dominantBottomColor(empty) = %#v, want %#v", got, want)
	}
}

func TestRenderTextureCopyRect(t *testing.T) {
	got := grRenderTextureRect(10, 20, 30, 40, 480)
	if got.X != 10 || got.Y != 420 || got.Width != 30 || got.Height != -40 {
		t.Fatalf("grRenderTextureRect() = %#v, want {10 420 30 -40}", got)
	}
}

func TestEmbeddedDrawScreenOpcodesUseSupportedBuffers(t *testing.T) {
	resetEmbeddedResourcesForTest(t)
	parseResourceFiles("assets/RESOURCE.MAP")

	found := 0
	for resourceIndex := 0; resourceIndex < numTtmResources; resourceIndex++ {
		resource := &ttmResources[resourceIndex]
		data := resource.UncompressedData
		for offset := 0; offset+2 <= len(data); {
			instructionOffset := offset
			opcode := binary.LittleEndian.Uint16(data[offset:])
			offset += 2
			argCount := int(opcode & 0x000f)
			if argCount == 0x0f {
				for offset < len(data) && data[offset] != 0 {
					offset++
				}
				offset++
				if offset%2 != 0 {
					offset++
				}
				continue
			}
			if offset+argCount*2 > len(data) {
				t.Fatalf("%s has truncated opcode %#04x at %#x", resource.ResName, opcode, instructionOffset)
			}
			if opcode == 0xB606 {
				found++
				args := make([]uint16, argCount)
				for i := range args {
					args[i] = binary.LittleEndian.Uint16(data[offset+i*2:])
				}
				if len(args) != 6 || args[4] > ttmBufferComposition || args[5] > ttmBufferComposition {
					t.Errorf("%s DRAW_SCREEN at %#x has unsupported args %v", resource.ResName, instructionOffset, args)
				}
				t.Logf("%s DRAW_SCREEN at %#x: %v", resource.ResName, instructionOffset, args)
			}
			offset += argCount * 2
		}
	}
	if found == 0 {
		t.Fatal("embedded TTMs contain no DRAW_SCREEN instructions")
	}
}

func TestRotateLogIfNeeded(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, appLogName)
	oldBackup := []byte("old backup")
	current := []byte("current log at the size limit")
	if err := os.WriteFile(logPath+".1", oldBackup, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, current, 0644); err != nil {
		t.Fatal(err)
	}

	if err := rotateLogIfNeeded(logPath, int64(len(current))); err != nil {
		t.Fatalf("rotateLogIfNeeded() error = %v", err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("active log still exists after rotation: %v", err)
	}
	got, err := os.ReadFile(logPath + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(current) {
		t.Fatalf("backup = %q, want %q", got, current)
	}
}

func TestAllShortScreensHaveValidPaddingColor(t *testing.T) {
	resetEmbeddedResourcesForTest(t)
	parseResourceFiles("assets/RESOURCE.MAP")
	grLoadPalette(&palResources[0])

	shortScreens := 0
	for index := 0; index < numScrResources; index++ {
		resource := &scrResources[index]
		if resource.Height >= screenHeight {
			continue
		}
		shortScreens++
		pixels, width, height := screenPixelData(resource)
		fill := dominantBottomColor(pixels, width, height)
		if fill.A != 255 {
			t.Errorf("%s padding color alpha = %d, want 255", resource.ResName, fill.A)
		}
		t.Logf("%s: %dx%d, padding RGBA(%d,%d,%d,%d)", resource.ResName, width, height, fill.R, fill.G, fill.B, fill.A)
	}
	if shortScreens == 0 {
		t.Fatal("embedded resources contain no short screens to validate")
	}
}

func resetEmbeddedResourcesForTest(t *testing.T) {
	if err := loadResourceArchives("assets"); err != nil {
		t.Skipf("original user-supplied test data is unavailable: %v", err)
	}
	oldADSResources, oldADSCount := adsResources, numAdsResources
	oldBMPResources, oldBMPCount := bmpResources, numBmpResources
	oldPALResources, oldPALCount := palResources, numPalResources
	oldSCRResources, oldSCRCount := scrResources, numScrResources
	oldTTMResources, oldTTMCount := ttmResources, numTtmResources
	t.Cleanup(func() {
		adsResources, numAdsResources = oldADSResources, oldADSCount
		bmpResources, numBmpResources = oldBMPResources, oldBMPCount
		scrResources, numScrResources = oldSCRResources, oldSCRCount
		palResources, numPalResources = oldPALResources, oldPALCount
		ttmResources, numTtmResources = oldTTMResources, oldTTMCount
	})

	adsResources = make([]TAdsResource, MaxADSResources)
	bmpResources = make([]TBMPResource, MaxBMPResources)
	scrResources = make([]TSCRResource, MaxSCRResources)
	palResources = make([]TPALResource, MaxPALResources)
	ttmResources = make([]TTTMResource, MaxTTMResources)
	numAdsResources, numBmpResources, numPalResources, numScrResources, numTtmResources = 0, 0, 0, 0, 0
}

func TestEmbeddedArchiveHashesAndDecompression(t *testing.T) {
	if err := loadResourceArchives("assets"); err != nil {
		t.Skipf("original user-supplied test data is unavailable: %v", err)
	}
	tests := []struct {
		name string
		data []byte
		md5  string
	}{
		{name: "RESOURCE.MAP", data: resourceMapData, md5: canonicalMapMD5},
		{name: "RESOURCE.001", data: resourceArchiveData, md5: canonicalArchiveMD5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := fmt.Sprintf("%x", md5.Sum(test.data))
			if got != test.md5 {
				t.Fatalf("MD5 = %s, want verified archive %s", got, test.md5)
			}
		})
	}

	resetEmbeddedResourcesForTest(t)
	parseResourceFiles("assets/RESOURCE.MAP")
	if numAdsResources != 10 || numBmpResources != 117 || numPalResources != 1 || numScrResources != 10 || numTtmResources != 41 {
		t.Fatalf("parsed resource counts ADS=%d BMP=%d PAL=%d SCR=%d TTM=%d", numAdsResources, numBmpResources, numPalResources, numScrResources, numTtmResources)
	}
	for index := 0; index < numAdsResources; index++ {
		resource := &adsResources[index]
		if len(resource.UncompressedData) != int(resource.UncompressedSize) {
			t.Errorf("%s decompressed to %d bytes, want %d", resource.ResName, len(resource.UncompressedData), resource.UncompressedSize)
		}
	}
	for index := 0; index < numBmpResources; index++ {
		resource := &bmpResources[index]
		if len(resource.UncompressedData) != int(resource.UncompressedSize) {
			t.Errorf("%s decompressed to %d bytes, want %d", resource.ResName, len(resource.UncompressedData), resource.UncompressedSize)
		}
	}
	for index := 0; index < numScrResources; index++ {
		resource := &scrResources[index]
		if len(resource.UncompressedData) != int(resource.UncompressedSize) {
			t.Errorf("%s decompressed to %d bytes, want %d", resource.ResName, len(resource.UncompressedData), resource.UncompressedSize)
		}
	}
	for index := 0; index < numTtmResources; index++ {
		resource := &ttmResources[index]
		if len(resource.UncompressedData) != int(resource.UncompressedSize) {
			t.Errorf("%s decompressed to %d bytes, want %d", resource.ResName, len(resource.UncompressedData), resource.UncompressedSize)
		}
	}
}

func TestMalformedRLEFailsClearly(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		inSize  uint32
		outSize uint32
	}{
		{name: "truncated literal", input: []byte{2, 0xAA}, inSize: 2, outSize: 2},
		{name: "run exceeds output", input: []byte{0x82, 0xAA}, inSize: 2, outSize: 1},
		{name: "declared input exceeds data", input: []byte{0x81, 0xAA}, inSize: 3, outSize: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				message := fmt.Sprint(recovered)
				if recovered == nil || !bytes.Contains([]byte(message), []byte("invalid RLE compressed resource")) {
					t.Fatalf("panic = %q, want clear RLE validation error", message)
				}
			}()
			uncompressRLE(bytes.NewReader(test.input), test.inSize, test.outSize)
		})
	}
}

func TestMalformedLZWFailsClearly(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		inSize  uint32
		outSize uint32
	}{
		{name: "declared input exceeds data", input: []byte{0}, inSize: 2, outSize: 1},
		{name: "oversized output", input: []byte{0}, inSize: 1, outSize: maxUncompressedResourceSize + 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				recovered := recover()
				message := fmt.Sprint(recovered)
				if recovered == nil || !bytes.Contains([]byte(message), []byte("invalid LZW compressed resource")) {
					t.Fatalf("panic = %q, want clear LZW validation error", message)
				}
			}()
			uncompressLZW(bytes.NewReader(test.input), test.inSize, test.outSize)
		})
	}
}

func TestPersistentSettingsConfigRoundTrip(t *testing.T) {
	want := TConfig{
		CurrentDay:       7,
		CurrentDate:      196,
		CRTFilterEnabled: true,
		CRTMode:          "lottes",
		SmoothingEnabled: true,
		ScalingMode:      "scale2x",
		ScenesInOrder:    true,
		Windowed:         true,
		Mute:             true,
		Stretch:          true,
		Monitor:          2,
		FastCRTSharpness: 3,
		ShowPerformance:  true,
	}
	var got TConfig
	for _, line := range strings.Split(strings.TrimSpace(cfgFormat(&want)), "\n") {
		if err := cfgApplyLine(&got, line); err != nil {
			t.Fatalf("cfgApplyLine(%q): %v", line, err)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("config round trip = %#v, want %#v", got, want)
	}
}

func TestMergePersistentAppOptions(t *testing.T) {
	cfg := TConfig{Windowed: true, Mute: true, Stretch: true, Monitor: 3}
	got := mergePersistentAppOptions(appOptions{monitor: 1}, cfg, nil)
	want := appOptions{windowed: true, mute: true, stretch: true, monitor: 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged options = %#v, want %#v", got, want)
	}

	overridden := mergePersistentAppOptions(
		appOptions{monitor: 2}, cfg,
		[]string{"--fullscreen", "--sound", "--fit", "--monitor=2"},
	)
	if overridden.windowed || overridden.mute || overridden.stretch || overridden.monitor != 2 {
		t.Fatalf("explicit overrides were replaced: %#v", overridden)
	}

	screensaver := mergePersistentAppOptions(appOptions{screenSaver: true, monitor: 1}, cfg, nil)
	if screensaver.windowed {
		t.Fatal("screensaver inherited persisted windowed mode")
	}
}

func TestScalingModeMigrationAndLabels(t *testing.T) {
	if got := parseImageScalingMode("", true); got != scalingBilinear {
		t.Fatalf("legacy smoothing migrated to %q, want bilinear", got)
	}
	if got := parseImageScalingMode("scale2x", false); got != scalingScale2x {
		t.Fatalf("scaling mode = %q, want scale2x", got)
	}
	if got := parseImageScalingMode("invalid", false); got != scalingNearest {
		t.Fatalf("invalid scaling mode = %q, want nearest", got)
	}
}

func TestCRTFilterMigrationAndLabels(t *testing.T) {
	if got := parseCRTFilter("", true); got != crtLightweight {
		t.Fatalf("legacy CRT setting migrated to %q, want lightweight", got)
	}
	if got := parseCRTFilter("lottes", false); got != crtLottes {
		t.Fatalf("CRT mode = %q, want lottes", got)
	}
	if got := parseCRTFilter("fast", false); got != crtFast {
		t.Fatalf("CRT mode = %q, want fast", got)
	}
	if got := parseCRTFilter("invalid", false); got != crtOff {
		t.Fatalf("invalid CRT mode = %q, want off", got)
	}
	if crtLottes.label() != "Lottes" {
		t.Fatalf("Lottes label = %q", crtLottes.label())
	}
}

func TestFastCRTSharpnessPersistence(t *testing.T) {
	if got := decodeFastCRTSharpness(0); got != fastCRTBalanced {
		t.Fatalf("missing sharpness decoded as %d, want balanced", got)
	}
	for mode := fastCRTSoft; mode <= fastCRTSharp; mode++ {
		if got := decodeFastCRTSharpness(encodeFastCRTSharpness(mode)); got != mode {
			t.Errorf("sharpness %d round trip = %d", mode, got)
		}
	}
	if fastCRTSharpnessValue(fastCRTSoft) >= fastCRTSharpnessValue(fastCRTSharp) {
		t.Fatal("soft sharpness is not lower than sharp preset")
	}
}

func TestPerformanceImpactAndCapacity(t *testing.T) {
	if got := int(math.Round(performanceBudgetPercent(10))); got != 30 {
		t.Fatalf("10 ms budget = %d%%, want 30%%", got)
	}
	tests := []struct {
		ms   float64
		want string
	}{
		{ms: 2, want: "Very low"},
		{ms: 8, want: "Low"},
		{ms: 16, want: "Moderate"},
		{ms: 25, want: "High"},
		{ms: 31, want: "Critical"},
	}
	for _, test := range tests {
		if got := performanceImpactLabel(test.ms); got != test.want {
			t.Errorf("impact at %.1f ms = %q, want %q", test.ms, got, test.want)
		}
	}
}

func TestPerformanceBenchmarkModeOrder(t *testing.T) {
	want := []crtFilter{crtOff, crtLightweight, crtFast, crtLottes}
	if !reflect.DeepEqual(performanceBenchmarkModes, want) {
		t.Fatalf("benchmark modes = %#v, want %#v", performanceBenchmarkModes, want)
	}
}

func TestScreensaverExitPolicy(t *testing.T) {
	if !shouldExitScreenSaver(true, false, false) {
		t.Fatal("mouse movement outside an overlay should exit")
	}
	if shouldExitScreenSaver(true, true, false) {
		t.Fatal("mouse movement over an interactive overlay should not exit")
	}
	if !shouldExitScreenSaver(false, true, true) {
		t.Fatal("unlisted keyboard input should exit even with an overlay open")
	}
	if shouldExitScreenSaver(false, false, false) {
		t.Fatal("screensaver exited without input")
	}
}

func TestDoubleEscapeQuitWindow(t *testing.T) {
	if doubleEscapeShouldExit(false, 0, 0.5) {
		t.Fatal("an unarmed Escape press requested exit")
	}
	if !doubleEscapeShouldExit(true, 10, 11.5) {
		t.Fatal("second Escape at the window boundary did not request exit")
	}
	if doubleEscapeShouldExit(true, 10, 11.5001) {
		t.Fatal("expired double-Escape sequence requested exit")
	}
	if doubleEscapeShouldExit(true, 10, 9) {
		t.Fatal("time moving backwards requested exit")
	}
}

func TestScreensaverFooterOpacity(t *testing.T) {
	tests := []struct {
		elapsed float64
		want    float32
	}{
		{elapsed: 0, want: 1},
		{elapsed: 8, want: 1},
		{elapsed: 9, want: 0.5},
		{elapsed: 10, want: 0},
		{elapsed: 12, want: 0},
	}
	for _, test := range tests {
		if got := menuFooterOpacity(test.elapsed); got != test.want {
			t.Errorf("menuFooterOpacity(%v) = %v, want %v", test.elapsed, got, test.want)
		}
	}
}

func TestDisplayViewportAt32By9Resolutions(t *testing.T) {
	tests := []struct {
		width, height       float32
		wantX, wantW, wantH float32
	}{
		{width: 5120, height: 1440, wantX: 1600, wantW: 1920, wantH: 1440},
		{width: 7680, height: 2160, wantX: 2400, wantW: 2880, wantH: 2160},
	}
	for _, test := range tests {
		viewport := calculateDisplayViewport(test.width, test.height, false)
		if viewport.x != test.wantX || viewport.y != 0 || viewport.width != test.wantW || viewport.height != test.wantH {
			t.Errorf("viewport for %.0fx%.0f = %#v", test.width, test.height, viewport)
		}
	}
	stretched := calculateDisplayViewport(5120, 1440, true)
	if stretched != (displayViewport{width: 5120, height: 1440}) {
		t.Fatalf("stretched viewport = %#v", stretched)
	}
}

func TestMonitorSelectionAnd32By9Detection(t *testing.T) {
	if got := resolveMonitorIndex(2, 3); got != 1 {
		t.Fatalf("monitor index = %d, want 1", got)
	}
	if got := resolveMonitorIndex(4, 3); got != 0 {
		t.Fatalf("unavailable monitor index = %d, want fallback 0", got)
	}
	if !is32By9Display(5120, 1440) || !is32By9Display(7680, 2160) {
		t.Fatal("expected both Neo G9-class resolutions to be detected as 32:9")
	}
	if is32By9Display(3840, 2160) {
		t.Fatal("16:9 display detected as 32:9")
	}
}

func TestDecodeADSScript(t *testing.T) {
	data := make([]byte, 0, 10)
	data = binary.LittleEndian.AppendUint16(data, 7)
	data = binary.LittleEndian.AppendUint16(data, 0x1360)
	data = binary.LittleEndian.AppendUint16(data, 1)
	data = binary.LittleEndian.AppendUint16(data, 2)
	data = binary.LittleEndian.AppendUint16(data, 0xFFFF)

	lines := decodeADSScript(data)
	if len(lines) != 3 {
		t.Fatalf("decodeADSScript() returned %d lines, want 3", len(lines))
	}
	if lines[0].offset != 0 || lines[0].instruction != "TAG 7" {
		t.Fatalf("tag line = %#v", lines[0])
	}
	if lines[1].offset != 2 || lines[1].instruction != "IF_NOT_RUNNING 1 2" {
		t.Fatalf("condition line = %#v", lines[1])
	}
	if lines[2].offset != 8 || lines[2].instruction != "END" {
		t.Fatalf("end line = %#v", lines[2])
	}
}

func TestDecodeEveryEmbeddedADSScript(t *testing.T) {
	resetEmbeddedResourcesForTest(t)
	parseResourceFiles("assets/RESOURCE.MAP")
	for index := 0; index < numAdsResources; index++ {
		resource := &adsResources[index]
		lines := decodeADSScript(resource.UncompressedData)
		if len(lines) == 0 {
			t.Errorf("%s decoded to no ADS instructions", resource.ResName)
			continue
		}
		previous := uint32(0)
		for lineIndex, line := range lines {
			if lineIndex > 0 && line.offset <= previous {
				t.Errorf("%s offsets are not increasing at line %d", resource.ResName, lineIndex)
			}
			if strings.Contains(line.instruction, "<truncated>") {
				t.Errorf("%s has truncated instruction at offset %#x", resource.ResName, line.offset)
			}
			previous = line.offset
		}
	}
}

func TestEmbeddedADSConditionInventory(t *testing.T) {
	resetEmbeddedResourcesForTest(t)
	parseResourceFiles("assets/RESOURCE.MAP")
	conditionCount := 0
	for index := 0; index < numAdsResources; index++ {
		resource := &adsResources[index]
		lines := decodeADSScript(resource.UncompressedData)
		for lineIndex, line := range lines {
			if line.opcode < 0x1300 || line.opcode > 0x1430 {
				continue
			}
			conditionCount++
			t.Logf("%s: %s", resource.ResName, line.instruction)
			if line.opcode == 0x1420 || line.opcode == 0x1430 {
				if lineIndex == 0 || lineIndex+1 >= len(lines) {
					t.Errorf("%s has dangling %s", resource.ResName, line.instruction)
					continue
				}
				next := lines[lineIndex+1].opcode
				if next < 0x1300 || next >= 0x1400 {
					t.Errorf("%s %s is followed by %s", resource.ResName, line.instruction, lines[lineIndex+1].instruction)
				}
			}
		}
	}
	if conditionCount == 0 {
		t.Fatal("embedded ADS files contain no conditions")
	}
}

func TestEmbeddedADSLocalControlInventory(t *testing.T) {
	resetEmbeddedResourcesForTest(t)
	parseResourceFiles("assets/RESOURCE.MAP")
	found := 0
	for index := 0; index < numAdsResources; index++ {
		resource := &adsResources[index]
		lines := decodeADSScript(resource.UncompressedData)
		for lineIndex, line := range lines {
			if line.opcode != 0x1070 {
				continue
			}
			found++
			if lineIndex+2 >= len(lines) {
				t.Fatalf("%s has truncated WHILE_RUNNING block", resource.ResName)
			}
			if line.instruction != "WHILE_RUNNING 4 5" ||
				lines[lineIndex+1].instruction != "END_WHILE" ||
				lines[lineIndex+2].instruction != "ADD_SCENE 4 22 0 1" {
				t.Errorf("%s local wait decoded as %q / %q / %q", resource.ResName, line.instruction, lines[lineIndex+1].instruction, lines[lineIndex+2].instruction)
			}
		}
	}
	if found != 1 {
		t.Fatalf("found %d WHILE_RUNNING blocks, want 1", found)
	}
}

func TestADSConditionState(t *testing.T) {
	t.Run("AND remains false", func(t *testing.T) {
		var state adsConditionState
		state.add(true)
		state.setOperator(0x1420)
		state.add(false)
		state.setOperator(0x1420)
		state.add(true)
		if state.shouldExecute() {
			t.Fatal("true AND false AND true evaluated true")
		}
	})

	t.Run("OR accepts either branch", func(t *testing.T) {
		var state adsConditionState
		state.add(false)
		state.setOperator(0x1430)
		state.add(true)
		if !state.shouldExecute() {
			t.Fatal("false OR true evaluated false")
		}
	})

	t.Run("reset removes skipped block", func(t *testing.T) {
		var state adsConditionState
		state.add(false)
		state.reset()
		if !state.shouldExecute() {
			t.Fatal("reset condition still skips operations")
		}
	})
}

func TestADSConditionBlocksControlOperations(t *testing.T) {
	oldThreads, oldNumThreads := ttmThreads, numThreads
	oldStopRequested, oldPlayed := adsStopRequested, adsPlayedScenes
	t.Cleanup(func() {
		ttmThreads, numThreads = oldThreads, oldNumThreads
		adsStopRequested, adsPlayedScenes = oldStopRequested, oldPlayed
	})

	appendInstruction := func(data []byte, opcode uint16, args ...uint16) []byte {
		data = binary.LittleEndian.AppendUint16(data, opcode)
		for _, arg := range args {
			data = binary.LittleEndian.AppendUint16(data, arg)
		}
		return data
	}

	t.Run("failed AND skips stop", func(t *testing.T) {
		ttmThreads = [MaxTTMThreads]TTtmThread{}
		ttmThreads[0] = TTtmThread{isRunning: 1, sceneSlot: 1, sceneTag: 1}
		numThreads = 1
		data := appendInstruction(nil, 0x1360, 1, 1) // false
		data = appendInstruction(data, 0x1420)
		data = appendInstruction(data, 0x1360, 2, 2) // true
		data = appendInstruction(data, 0x2010, 1, 1, 0)
		data = appendInstruction(data, 0x1510)
		data = appendInstruction(data, 0xFFFF)
		adsPlayChunk(data, uint32(len(data)), 0)
		if ttmThreads[0].isRunning == 0 {
			t.Fatal("STOP_SCENE ran for a false AND condition")
		}
	})

	t.Run("successful OR runs stop", func(t *testing.T) {
		ttmThreads = [MaxTTMThreads]TTtmThread{}
		ttmThreads[0] = TTtmThread{isRunning: 1, sceneSlot: 1, sceneTag: 1}
		numThreads = 1
		data := appendInstruction(nil, 0x1370, 1, 1) // true
		data = appendInstruction(data, 0x1430)
		data = appendInstruction(data, 0x1370, 2, 2) // false
		data = appendInstruction(data, 0x2010, 1, 1, 0)
		data = appendInstruction(data, 0x1510)
		adsPlayChunk(data, uint32(len(data)), 0)
		if ttmThreads[0].isRunning != 0 {
			t.Fatal("STOP_SCENE was skipped for a true OR condition")
		}
	})
}

func TestTraceFilteringSearchAndADSHighlight(t *testing.T) {
	oldHistory, oldFilter, oldSearch := traceHistory, traceCurrentFilter, traceSearch
	oldView, oldADSName, oldADSLines := traceCurrentView, traceADSName, traceADSLines
	oldADSOffset, oldADSActive := traceADSActiveOffset, traceADSHasActive
	t.Cleanup(func() {
		traceHistory, traceCurrentFilter, traceSearch = oldHistory, oldFilter, oldSearch
		traceCurrentView, traceADSName, traceADSLines = oldView, oldADSName, oldADSLines
		traceADSActiveOffset, traceADSHasActive = oldADSOffset, oldADSActive
	})

	traceHistory = []ttmTraceEntry{
		{sequence: 1, source: traceSourceTTM, script: "FIRE.TTM", instruction: "LOAD_IMAGE FLAME.BMP"},
		{sequence: 2, source: traceSourceADS, script: "BUILDING.ADS", instruction: "IF_NOT_RUNNING 0 1"},
	}
	traceCurrentFilter = traceFilterADS
	if got := traceHistoryEntries(); len(got) != 1 || got[0].source != traceSourceADS {
		t.Fatalf("ADS filter returned %#v", got)
	}
	traceCurrentFilter = traceFilterAll
	traceSearch = "flame"
	if got := traceHistoryEntries(); len(got) != 1 || got[0].source != traceSourceTTM {
		t.Fatalf("search returned %#v", got)
	}

	traceCurrentView = traceViewADSScript
	traceSearch = ""
	traceADSName = "BUILDING.ADS"
	traceADSLines = []adsTraceLine{{offset: 4, opcode: 0x1360, instruction: "IF_NOT_RUNNING 0 1"}}
	traceADSActiveOffset = 4
	traceADSHasActive = true
	text := traceCurrentViewText()
	if !strings.Contains(text, "> 0004  1360  IF_NOT_RUNNING 0 1") {
		t.Fatalf("ADS export lacks active marker: %q", text)
	}
}

func TestAppVersionLabel(t *testing.T) {
	if label := appVersionLabel(); !strings.Contains(label, appVersion) || !strings.Contains(label, appBuildIdentifier()) {
		t.Fatalf("appVersionLabel() = %q", label)
	}
}
