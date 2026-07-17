package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var screenshotRequested bool

type pngTextTag struct {
	key   string
	value string
}

var pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}

func screenshotFilename(now time.Time) string {
	return "Johnny-Castaway-" + now.Format("20060102-150405.000") + ".png"
}

func requestScreenshot() {
	screenshotRequested = true
}

func screenshotMetadata(now time.Time) []pngTextTag {
	content := currentContent
	if content == "" {
		content = "Full Story"
	}
	windowMode := "Fullscreen"
	if appSettings.previewParent != 0 {
		windowMode = "Screensaver preview"
	} else if appSettings.screenSaver {
		windowMode = "Screensaver"
	} else if appSettings.windowed {
		windowMode = "Windowed"
	}
	aspect := "Fit 4:3"
	if appSettings.stretch {
		aspect = "Stretch"
	}
	return []pngTextTag{
		{key: "Title", value: "Johnny Castaway"},
		{key: "Software", value: appVersionLabel()},
		{key: "Creation Time", value: now.Format(time.RFC3339)},
		{key: "Display Filter", value: crtFilterMode.label()},
		{key: "CRT Sharpness", value: fastCRTSharpnessLabel(fastCRTSharpness)},
		{key: "Image Scaling", value: imageScalingMode.label()},
		{key: "Aspect Mode", value: aspect},
		{key: "Scene Order", value: storyPlaybackModeLabel()},
		{key: "Content", value: content},
		{key: "Sky", value: islandDayNightLabel()},
		{key: "Holiday", value: islandHolidayLabel()},
		{key: "Window Mode", value: windowMode},
		{key: "Resolution", value: fmt.Sprintf("%dx%d", rl.GetScreenWidth(), rl.GetScreenHeight())},
		{key: "Muted", value: strconv.FormatBool(appSettings.mute)},
		{key: "Paused", value: strconv.FormatBool(playbackPaused)},
	}
}

func makePNGTextChunk(tag pngTextTag) ([]byte, error) {
	if len(tag.key) == 0 || len(tag.key) > 79 || strings.ContainsRune(tag.key, 0) {
		return nil, fmt.Errorf("invalid PNG metadata key %q", tag.key)
	}
	if strings.ContainsRune(tag.value, 0) {
		return nil, fmt.Errorf("PNG metadata %q contains a null byte", tag.key)
	}
	payload := append(append([]byte(tag.key), 0), []byte(tag.value)...)
	chunk := make([]byte, 12+len(payload))
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(payload)))
	copy(chunk[4:8], "tEXt")
	copy(chunk[8:8+len(payload)], payload)
	binary.BigEndian.PutUint32(chunk[8+len(payload):], crc32.ChecksumIEEE(chunk[4:8+len(payload)]))
	return chunk, nil
}

func annotatePNG(data []byte, tags []pngTextTag) ([]byte, error) {
	if len(data) < len(pngSignature) || !bytes.Equal(data[:len(pngSignature)], pngSignature) {
		return nil, fmt.Errorf("invalid PNG signature")
	}
	result := append([]byte(nil), pngSignature...)
	position := len(pngSignature)
	inserted := false
	for position < len(data) {
		if len(data)-position < 12 {
			return nil, fmt.Errorf("truncated PNG chunk header")
		}
		length := uint64(binary.BigEndian.Uint32(data[position : position+4]))
		chunkSize := uint64(12) + length
		if chunkSize > uint64(len(data)-position) {
			return nil, fmt.Errorf("truncated PNG chunk payload")
		}
		end := position + int(chunkSize)
		chunkType := string(data[position+4 : position+8])
		result = append(result, data[position:end]...)
		position = end
		if chunkType == "IHDR" && !inserted {
			for _, tag := range tags {
				chunk, err := makePNGTextChunk(tag)
				if err != nil {
					return nil, err
				}
				result = append(result, chunk...)
			}
			inserted = true
		}
		if chunkType == "IEND" {
			if position != len(data) {
				return nil, fmt.Errorf("unexpected data after PNG end")
			}
			break
		}
	}
	if !inserted {
		return nil, fmt.Errorf("PNG header chunk is missing")
	}
	return result, nil
}

func pngTextMetadata(data []byte) (map[string]string, error) {
	if len(data) < len(pngSignature) || !bytes.Equal(data[:len(pngSignature)], pngSignature) {
		return nil, fmt.Errorf("invalid PNG signature")
	}
	metadata := make(map[string]string)
	for position := len(pngSignature); position < len(data); {
		if len(data)-position < 12 {
			return nil, fmt.Errorf("truncated PNG chunk header")
		}
		length := int(binary.BigEndian.Uint32(data[position : position+4]))
		end := position + 12 + length
		if length < 0 || end < position || end > len(data) {
			return nil, fmt.Errorf("truncated PNG chunk payload")
		}
		if string(data[position+4:position+8]) == "tEXt" {
			payload := data[position+8 : position+8+length]
			separator := bytes.IndexByte(payload, 0)
			if separator <= 0 {
				return nil, fmt.Errorf("invalid PNG text chunk")
			}
			metadata[string(payload[:separator])] = string(payload[separator+1:])
		}
		position = end
	}
	return metadata, nil
}

func captureScreenshot() (string, error) {
	directory, err := picturesDirectory()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", fmt.Errorf("create screenshot folder: %w", err)
	}
	now := time.Now()
	path := filepath.Join(directory, screenshotFilename(now))
	rl.TakeScreenshot(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("save screenshot: %w", err)
	}
	annotated, err := annotatePNG(data, screenshotMetadata(now))
	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("add screenshot metadata: %w", err)
	}
	if err := os.WriteFile(path, annotated, 0o644); err != nil {
		return "", fmt.Errorf("write screenshot metadata: %w", err)
	}
	return path, nil
}
