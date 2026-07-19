package main

var (
	ttmDx = 0
	ttmDy = 0
)

type standaloneTTMSpritePreload struct {
	slot     uint16
	resource string
}

// Some original TTM tags are continuations that expect sprite slots populated
// by an earlier part of the story. Direct TTM playback has no earlier scene, so
// seed only those documented dependencies before running or selecting tags.
var standaloneTTMSpritePreloads = map[string][]standaloneTTMSpritePreload{
	"GJGULIVR.TTM": {
		{slot: 1, resource: "LILIPUTS.BMP"},
	},
	"GJLILIPU.TTM": {
		{slot: 2, resource: "STNDLAY.BMP"},
		{slot: 3, resource: "SLEEP.BMP"},
	},
}

// The ADS controllers sometimes compose one visible event from several TTM
// tags. Direct TTM playback has no ADS controller, so recreate the confirmed
// companion layers for tags whose original script explicitly starts them
// together and waits for the selected tag.
var standaloneTTMCompanionScenes = map[string]map[uint16][]uint16{
	"GJGULIVR.TTM": {
		15: []uint16{14},
		12: []uint16{58},
		52: []uint16{58},
		53: []uint16{58},
		54: []uint16{58},
		9:  []uint16{58, 12},
		61: []uint16{58, 66},
		66: []uint16{58, 61},
		68: []uint16{65},
		65: []uint16{68},
		63: []uint16{68},
		67: []uint16{60},
	},
	"GJGULL1.TTM": {
		26: []uint16{41},
		46: []uint16{35},
	},
	"GJLILIPU.TTM": {
		20: []uint16{24},
		24: []uint16{20},
		25: []uint16{20},
		26: []uint16{20, 25},
	},
	"GJVIS3.TTM": {
		52: []uint16{44},
		53: []uint16{44},
		60: []uint16{44},
	},
	"GJVIS5.TTM": {
		8: []uint16{1},
		7: []uint16{1, 9},
		9: []uint16{1, 10},
		3: []uint16{1},
	},
	"GJVIS5W.TTM": {
		3: []uint16{1},
	},
	"GJVIS6.TTM": {
		3: []uint16{4},
		5: []uint16{4, 9},
		9: []uint16{4, 6},
		7: []uint16{4},
		8: []uint16{4},
	},
	"GJDIVE.TTM": {
		13: []uint16{12},
		8:  []uint16{12},
		7:  []uint16{12},
		9:  []uint16{12},
		14: []uint16{12, 11},
	},
	"GJNAT3.TTM": {
		16: []uint16{18},
	},
	"GJNAT1.TTM": {
		20: []uint16{19},
		22: []uint16{21},
		25: []uint16{34},
	},
	"MJCOCO.TTM": {
		18: []uint16{17, 33},
		20: []uint16{19},
	},
	"MJCOCO1.TTM": {
		18: []uint16{17},
		20: []uint16{19},
	},
	"MJRAFT.TTM": {
		3:  []uint16{9},
		4:  []uint16{9},
		7:  []uint16{9},
		6:  []uint16{9},
		15: []uint16{9},
	},
	"MJFISH.TTM": {
		18: []uint16{1},
		44: []uint16{1, 43},
	},
	"MJFISHC.TTM": {
		17: []uint16{43},
		58: []uint16{43, 62},
	},
	"MJTELE.TTM": {
		12: []uint16{27},
		13: []uint16{27},
	},
	"MJREAD.TTM": {
		98:  []uint16{113},
		108: []uint16{113, 114},
		96:  []uint16{85},
		110: []uint16{1},
	},
	"MJBATH.TTM": {
		3:  []uint16{42, 20, 30, 19, 21},
		24: []uint16{42, 20, 30, 19, 14, 23},
	},
	"MJSAND.TTM": {
		5:  []uint16{1, 3},
		10: []uint16{1, 8},
		40: []uint16{1, 35},
		83: []uint16{1, 49, 53, 42},
		84: []uint16{1, 49, 57, 58, 59, 60},
		51: []uint16{1, 46, 79, 81, 82, 80, 50},
	},
	"MJFIRE.TTM": {
		40: []uint16{39, 80, 81},
		82: []uint16{141},
		48: []uint16{44, 144},
		49: []uint16{44, 144, 78},
	},
	"SJGLIMPS.TTM": {
		106: []uint16{105},
	},
	"SBREAKUP.TTM": {
		19: []uint16{33},
		28: []uint16{33},
		33: []uint16{37},
	},
	"SJLEAVES.TTM": {
		3: []uint16{2},
	},
	"SJWORK.TTM": {
		3: []uint16{9},
		4: []uint16{9},
		7: []uint16{9},
	},
	"SASKDATE.TTM": {
		111: []uint16{5},
		122: []uint16{116},
		145: []uint16{121, 135, 128, 150},
		141: []uint16{112, 138},
	},
	"SJMSSGE.TTM": {
		36: []uint16{29},
		19: []uint16{29, 6},
		21: []uint16{29, 6, 32},
		33: []uint16{29, 6, 32},
	},
	"THEEND.TTM": {
		1: []uint16{9},
		5: []uint16{9},
		8: []uint16{9, 4, 6},
	},
	"WOULDBE.TTM": {
		2:  []uint16{1, 4},
		14: []uint16{4},
		4:  []uint16{6},
		13: []uint16{6},
		7:  []uint16{6},
	},
}

func ttmLoadStandaloneSpriteDependencies(ttmSlot *TTtmSlot, name string) {
	for _, preload := range standaloneTTMSpritePreloads[name] {
		grLoadBmp(ttmSlot, preload.slot, preload.resource)
	}
}

func ttmStandaloneSceneSpriteResources(ttmSlot *TTtmSlot, name string, targetOffset uint32) map[uint16]string {
	resources := make(map[uint16]string)
	for _, preload := range standaloneTTMSpritePreloads[name] {
		resources[preload.slot] = preload.resource
	}
	if ttmSlot == nil {
		return resources
	}

	selectedSlot := uint16(0)
	data := ttmSlot.data
	limit := min(int(targetOffset), len(data))
	for offset := 0; offset+2 <= limit; {
		opcode := uint16(data[offset]) | uint16(data[offset+1])<<8
		offset += 2
		argCount := int(opcode & 0x000f)
		if argCount == 0x0f {
			start := offset
			for offset < limit && data[offset] != 0 {
				offset++
			}
			if offset >= limit {
				break
			}
			value := string(data[start:offset])
			offset++
			if offset%2 != 0 {
				offset++
			}
			if opcode == 0xF02F && selectedSlot < MaxBMPSlots {
				resources[selectedSlot] = value
			}
			continue
		}
		byteCount := argCount * 2
		if offset+byteCount > limit {
			break
		}
		if opcode == 0x1051 && argCount > 0 {
			selectedSlot = uint16(data[offset]) | uint16(data[offset+1])<<8
		}
		offset += byteCount
	}
	return resources
}

func ttmPrepareStandaloneSceneSprites(ttmSlot *TTtmSlot, name string, targetOffset uint32) {
	for slot, resource := range ttmStandaloneSceneSpriteResources(ttmSlot, name, targetOffset) {
		grLoadBmp(ttmSlot, slot, resource)
	}
}

// ttmFollowIslandPlacement keeps scene-owned drawing aligned with the island.
// Direct TTM selection must replace any temporary left-island correction left
// by the previous story scene.
func ttmFollowIslandPlacement(extraX int) {
	ttmDx = islandState.xPos + extraX
	ttmDy = islandState.yPos
}

func ttmUseStandalonePlacement() {
	// A directly selected TTM loads its own fixed background rather than the
	// randomized Full Story island. Reset both origins so its scene and holiday
	// layer follow that newly loaded background.
	islandState.xPos = 0
	islandState.yPos = 0
	ttmFollowIslandPlacement(0)
}

func ttmFindPreviousTag(ttmSlot *TTtmSlot, offset uint32) uint32 {
	var result uint32 = 0
	i := 0

	for i < ttmSlot.numTags && ttmSlot.tags[i].offset < offset {
		result = ttmSlot.tags[i].offset
		i++
	}

	return result
}

func ttmFindTag(ttmSlot *TTtmSlot, reqdTag uint16) uint32 {
	var result uint32 = 0
	i := 0

	for result == 0 && i < ttmSlot.numTags {
		if ttmSlot.tags[i].id == reqdTag {
			result = ttmSlot.tags[i].offset
		} else {
			i++
		}
	}

	if result == 0 {
		debugPrintf("WARN: TTM tag %d not found, returning offset 0000\n", reqdTag)
	}

	return result
}

func ttmLoadTTM(ttmSlot *TTtmSlot, name string) {
	ttmResource := findTTMResource(name)

	ttmSlot.name = name
	ttmSlot.data = ttmResource.UncompressedData
	ttmSlot.dataSize = ttmResource.UncompressedSize
	ttmSlot.numTags = int(ttmResource.NumTags)
	ttmSlot.tags = make([]TTtmTag, ttmSlot.numTags)

	// we have to bookmark every tag for later jumps
	offset := uint32(0)
	tagNo := 0

	for offset < ttmSlot.dataSize {
		opCode := peekUint16(ttmSlot.data, &offset)

		if opCode == 0x1111 || opCode == 0x1101 {
			arg := peekUint16(ttmSlot.data, &offset)
			ttmSlot.tags[tagNo].id = arg
			ttmSlot.tags[tagNo].offset = offset
			tagNo++ // TODO
		} else {
			numArgs := uint8(opCode & 0x000f)

			if numArgs == 0x0f {
				for ttmSlot.data[offset] != 0 && ttmSlot.data[offset+1] != 0 {
					offset += 2
				}
				offset += 2
			} else {
				offset += uint32(numArgs) << 1
			}
		}
	}

	// TODO : in SASKDATE.TTM, num SET_SCENE != ttmResource->numTags
	for tagNo < ttmSlot.numTags {
		ttmSlot.tags[tagNo].id = 0 // TODO is this useful ?
		tagNo++
	}
}

func ttmInitSlot(ttmSlot *TTtmSlot) {
	ttmSlot.name = ""
	ttmSlot.data = nil
	ttmSlot.dataSize = 0
	ttmSlot.tags = nil
	ttmSlot.numTags = 0
	for i := 0; i < MaxBMPSlots; i++ {
		ttmSlot.numSprites[i] = 0
		for image := 0; image < MaxSpritesPerBMP; image++ {
			ttmSlot.sprites[i][image] = nil
		}
	}
}

func ttmResetSlot(ttmSlot *TTtmSlot) {
	for i := 0; i < MaxBMPSlots; i++ {
		if ttmSlot.numSprites[i] != 0 {
			grReleaseBmp(ttmSlot, uint16(i))
		}
	}
	ttmSlot.name = ""
	ttmSlot.data = nil
	ttmSlot.dataSize = 0
	ttmSlot.tags = nil
	ttmSlot.numTags = 0
}

func ttmPlay(ttmThread *TTtmThread) {
	var (
		offset       uint32
		opCode       uint16
		continueLoop = true
		args         [10]uint16
		strBytesArg  = make([]byte, 200)
		finalStr     = "" // added by me -- r.c.
	)

	grDx = ttmDx
	grDy = ttmDy

	ttmSlot := ttmThread.ttmSlot
	offset = ttmThread.ip
	data := ttmSlot.data

	for continueLoop {
		instructionOffset := offset
		finalStr = ""
		opCode = peekUint16(data, &offset)
		numArgs := uint8(opCode) & 0x000f

		if numArgs == 0x0f {
			// ✅: verified this null-terminated string parsing works - Ralph
			i := 0

			for data[offset] != 0 {
				strBytesArg[i] = data[offset]
				i++
				offset++
			}
			// r.c - here we have a complete string w/o null terminator (for Go)
			finalStr = string(strBytesArg[0:i])

			// r.c. - this captures the null terminator (we don't care about it)
			strBytesArg[i] = data[offset]
			i++
			offset++

			// r.c. - this just ensures we're always at an even byte, probably for historical reasons.
			if i&0x01 == 0x01 { // always read an even number of uint8s
				strBytesArg[i] = data[offset] // TODO
				i++
				offset++
			}
		} else {
			// args are numArgs words
			peekUint16Block(data, &offset, args[:], int(numArgs))
		}

		var traceArgs []uint16
		if numArgs != 0x0f {
			traceArgs = args[:numArgs]
		}
		traceRecordTTMInstruction(ttmThread, instructionOffset, opCode, traceArgs, finalStr)

		switch opCode {
		case 0x0080:
			debugPrintln("\tDRAW BACKGROUND")
		case 0x0110:
			debugPrintln("\tPURGE")
			if ttmThread.sceneTimer != 0 {
				ttmThread.nextGotoOffset = ttmFindPreviousTag(ttmSlot, offset)
			} else {
				ttmThread.isRunning = 2
			}
		case 0x0FF0:
			debugPrintln("\tUPDATE")
			continueLoop = false
		case 0x1021:
			var result uint16
			if args[0] > 4 {
				result = args[0]
			} else {
				result = 4
			}
			ttmThread.timer = result
			ttmThread.delay = result

			debugPrintf("\tSET DELAY => %d\n", result)
		case 0x1051:
			debugPrintf("\tSET BMP SLOT: slot:%d\n", args[0])
			ttmThread.selectedBmpSlot = uint8(args[0])
		case 0x1061:
			debugPrintf("\tSET_PALETTE_SLOT: slot:%d\n", args[0])
		case 0x1101:
			debugPrintf("\t:LOCAL_TAG %d", args[0])
		case 0x1111:
			// r.c. seems like some script animation marker possibly, perhaps used for debugging.
			debugPrintf("\t:TAG #%d ------------------------\n", args[0])
			if currentContent != "" && !standaloneEventActive() && ttmThread == &ttmThreads[0] {
				adsSetStandaloneCompanions(ttmSlot.name, args[0])
			}
		case 0x1121:
			// Selects the GETPUT buffer used by SAVE_GETPUT_REGION.
			// (see WOULDBE.TTM for a nice example)
			debugPrintf("\tSET_GETPUT_SLOT %d\n", args[0])
		case 0x1201:
			// ex TTM_UNKNOWN_2
			debugPrintf("\tGOTO_TAG %d\n", args[0])
			ttmThread.nextGotoOffset = ttmFindTag(ttmSlot, args[0])
		case 0x2002:
			debugPrintf("\tSET_COLORS %d %d\n", args[0], args[1])
			ttmThread.fgColor = uint8(args[0])
			ttmThread.bgColor = uint8(args[1])
		case 0x2012:
			// args always == (0,0)
			// at beginning of scenes, near LOAD_IMAGEs
			debugPrintf("\tSET_FRAME1 %d %d\n", args[0], args[1])
		case 0x2022:
			debugPrintf("\tTIMER %d %d\n", args[0], args[1])
			// Really, really not sure about this formula... but things
			// do work not so bad like that
			val := (args[0] + args[1]) / 2
			ttmThread.delay = val
			ttmThread.timer = val
		case 0x4004:
			debugPrintf("\tSET_CLIP_ZONE: %d %d %d %d\n", args[0], args[1], args[2], args[3])
			grSetClipZone(ttmThread.ttmLayer, int16(args[0]), int16(args[1]), int16(args[2]), int16(args[3]))
		case 0x4204:
			debugPrintf("\tSTORE_AREA: x:%d, y:%d, w:%d, h:%d\n", args[0], args[1], args[2], args[3])
			grCopyZoneToBg(ttmThread.ttmLayer, args[0], args[1], args[2], args[3])
		case 0x4214:
			// defines the zone to be redrawn at each update ?
			// but seems not used in the original
			debugPrintf("\tSAVE_GETPUT_REGION %d %d %d %d\n", args[0], args[1], args[2], args[3])
			grSaveImage1(ttmThread.ttmLayer, args[0], args[1], args[2], args[3])
		case 0xA002:
			debugPrintf("\tDRAW_PIXEL %d %d\n", args[0], args[1])
			grDrawPixel(ttmThread.ttmLayer, int16(args[0]), int16(args[1]), ttmThread.fgColor)
		case 0xA054:
			// only once, in GJGULIVR.TTM.txt
			debugPrintf("\tSAVE_ZONE %d %d %d %d\n", args[0], args[1], args[2], args[3])
			grSaveZone(ttmThread.ttmLayer, args[0], args[1], args[2], args[3])
		case 0xA064:
			// only once, in GJGULIVR.TTM.txt
			debugPrintf("\tWIPE_RIGHT_TO_LEFT %d %d %d %d\n", args[0], args[1], args[2], args[3])
			// r.c. if I enable this, the stupid copied zone, disappears too soon!!
			//grRestoreZone(ttmThread.ttmLayer, args[0], args[1], args[2], args[3])
		case 0xA0A4:
			debugPrintf("\tDRAW_LINE %d %d %d %d\n", args[0], args[1], args[2], args[3])
			grDrawLine(ttmThread.ttmLayer, int16(args[0]), int16(args[1]), int16(args[2]), int16(args[3]), ttmThread.fgColor)
		case 0xA104:
			debugPrintf("\tDRAW_RECT %d %d %d %d\n", args[0], args[1], args[2], args[3])
			grDrawRect(ttmThread.ttmLayer, int16(args[0]), int16(args[1]), args[2], args[3], ttmThread.fgColor)
		case 0xA404:
			debugPrintf("\tDRAW_CIRCLE %d %d %d %d\n", args[0], args[1], args[2], args[3])
			grDrawCircle(ttmThread.ttmLayer, int16(args[0]), int16(args[1]), args[2], args[3], ttmThread.fgColor, ttmThread.bgColor)
		case 0xA504:
			debugPrintf("\tDRAW_SPRITE x:%d y:%d sprtNo:%d imgNo:%d\n", args[0], args[1], args[2], args[3])
			grDrawSprite(ttmThread.ttmLayer, ttmThread.ttmSlot, int16(args[0]), int16(args[1]), args[2], args[3])
		case 0xA524:
			debugPrintf("\tDRAW_SPRITE_FLIP x:%d y:%d sprtNo:%d imgNo:%d\n", args[0], args[1], args[2], args[3])
			grDrawSpriteFlip(ttmThread.ttmLayer, ttmThread.ttmSlot, int16(args[0]), int16(args[1]), args[2], args[3])
		case 0xA601:
			debugPrintf("\tDRAW_GETPUT %d\n", args[0])
			grClearScreen(ttmThread.ttmLayer)
		case 0xB606:
			debugPrintf("\tDRAW_SCREEN x:%d y:%d w:%d h:%d buffer:%d->%d\n", args[0], args[1], args[2], args[3], args[4], args[5])
			grCopyBuffer(ttmThread, args[0], args[1], args[2], args[3], args[4], args[5])
		case 0xC051:
			debugPrintf("\tPLAY SAMPLE: sampleId:%d\n", args[0])
			soundPlay(args[0])
		case 0xF01F:
			debugPrintf("\tLOAD_SCREEN: %q\n", finalStr)
			screenName := islandStandaloneScreen(finalStr)
			grLoadScreen(screenName)
			if screenName == "NIGHT.SCR" && standaloneDayScreenName != "" {
				islandDrawStandaloneNightIsland()
			}
		case 0xF02F:
			debugPrintf("\tLOAD_IMAGE: %q\n", finalStr)
			grLoadBmp(ttmSlot, uint16(ttmThread.selectedBmpSlot), finalStr)
		case 0xF05F:
			debugPrintf("\tLOAD_PALETTE: %q\n", finalStr)
		}

		if offset >= ttmSlot.dataSize {
			ttmThread.isRunning = 2
			continueLoop = false
		}
	}

	ttmThread.ip = offset
}
