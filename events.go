package main

import (
	"encoding/binary"
	"fmt"
	"strings"
)

type standaloneADSEvent struct {
	adsName     string
	tag         uint16
	description string
	scene       TStoryScene
}

var (
	standaloneEventTarget = ""
	standaloneEventIndex  = -1
)

func adsEventTagUsesSlot(resource *TAdsResource, tag, slot uint16) bool {
	if resource == nil {
		return false
	}
	currentTag := uint16(0)
	data := resource.UncompressedData
	for offset := 0; offset+2 <= len(data); {
		opcode := binary.LittleEndian.Uint16(data[offset:])
		offset += 2
		argCount, known := adsOpcodeArgCounts[opcode]
		if !known {
			// ADS tag IDs are the only words outside the instruction table.
			currentTag = opcode
			continue
		}
		if offset+argCount*2 > len(data) {
			return false
		}
		if currentTag == tag && opcode == 0x2005 && argCount >= 1 && binary.LittleEndian.Uint16(data[offset:]) == slot {
			return true
		}
		offset += argCount * 2
	}
	return false
}

func standaloneADSEventsForTTM(ttmName string) []standaloneADSEvent {
	var events []standaloneADSEvent
	// storyScenes is the canonical original playback order. Build the filtered
	// collection from it instead of resource-map order.
	for _, scene := range storyScenes {
		var resource *TAdsResource
		for resourceIndex := 0; resourceIndex < numAdsResources; resourceIndex++ {
			candidate := &adsResources[resourceIndex]
			if strings.EqualFold(candidate.ResName, scene.adsName) {
				resource = candidate
				break
			}
		}
		if resource == nil {
			continue
		}
		var slots []uint16
		for _, dependency := range resource.Res {
			if strings.EqualFold(strings.TrimSpace(dependency.Name), ttmName) {
				slots = append(slots, dependency.ID)
			}
		}
		if len(slots) == 0 {
			continue
		}
		usesSelectedTTM := false
		for _, slot := range slots {
			usesSelectedTTM = usesSelectedTTM || adsEventTagUsesSlot(resource, uint16(scene.adsTagNo), slot)
		}
		if !usesSelectedTTM {
			continue
		}
		description := ""
		for _, tag := range resource.Tags {
			if tag.ID == uint16(scene.adsTagNo) {
				description = strings.TrimSpace(tag.Description)
				break
			}
		}
		if description == "" {
			description = fmt.Sprintf("%s event %d", strings.TrimSuffix(resource.ResName, ".ADS"), scene.adsTagNo)
		}
		events = append(events, standaloneADSEvent{
			adsName: resource.ResName, tag: uint16(scene.adsTagNo), description: description, scene: scene,
		})
	}
	return events
}

func standaloneEventActive() bool {
	return currentContent != "" && standaloneEventTarget == currentContent && standaloneEventIndex >= 0
}

func standaloneSelectedEvent(target string) (standaloneADSEvent, int, int, bool) {
	if target == "" || standaloneEventTarget != target || standaloneEventIndex < 0 {
		return standaloneADSEvent{}, 0, 0, false
	}
	events := standaloneADSEventsForTTM(target)
	if len(events) == 0 {
		return standaloneADSEvent{}, 0, 0, false
	}
	standaloneEventIndex %= len(events)
	return events[standaloneEventIndex], standaloneEventIndex + 1, len(events), true
}

func standaloneSelectNextEvent(target string) (standaloneADSEvent, int, int, bool) {
	events := standaloneADSEventsForTTM(target)
	if len(events) == 0 {
		return standaloneADSEvent{}, 0, 0, false
	}
	if standaloneEventTarget != target || standaloneEventIndex < 0 {
		standaloneEventTarget = target
		standaloneEventIndex = 0
	} else {
		standaloneEventIndex = (standaloneEventIndex + 1) % len(events)
	}
	return events[standaloneEventIndex], standaloneEventIndex + 1, len(events), true
}

func standaloneAdvanceEvent(target string) {
	events := standaloneADSEventsForTTM(target)
	if len(events) != 0 && standaloneEventTarget == target && standaloneEventIndex >= 0 {
		standaloneEventIndex = (standaloneEventIndex + 1) % len(events)
	}
}

func standaloneResetEventMode() {
	standaloneEventTarget = ""
	standaloneEventIndex = -1
}

func standaloneEventStatus(event standaloneADSEvent, number, total int) string {
	return fmt.Sprintf("Event %d/%d: %s (%s:%d)", number, total, event.description, event.adsName, event.tag)
}

func adsPlayStandaloneEvent(event standaloneADSEvent) {
	adsInit()
	storyCalculateIslandFromDateAndTime()
	if event.scene.flags&ISLAND != 0 {
		storyCalculateIslandFromScene(&event.scene)
		adsInitIsland()
		extraX := 0
		if event.scene.flags&LEFT_ISLAND != 0 {
			extraX = 272
		}
		ttmFollowIslandPlacement(extraX)
	} else {
		adsNoIsland()
		ttmDx, ttmDy = 0, 0
	}
	adsPlay(event.adsName, event.tag)
	if event.scene.flags&ISLAND != 0 {
		adsReleaseIsland()
	}
}
