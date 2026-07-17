package main

var (
	walkPath      []int
	walkPathIndex int
	currentSpot   int
	currentHdg    int
	nextSpot      int
	nextHdg       int
	finalSpot     int
	finalHdg      int
	increment     int
	lastTurn      int
	hasArrived    int
	isBehindTree  int

	walkDataIndex int
)

const (
	walkTreeTrunkSprite  = 13
	walkTreeLeavesSprite = 12
)

type walkDrawItem struct {
	sprite  uint16
	x, y    int16
	flipped bool
	tree    bool
}

func nextWalkPathSpot() int {
	walkPathIndex++
	if walkPathIndex >= len(walkPath) || walkPath[walkPathIndex] == UndefNode {
		panic("walk path ended before destination")
	}
	return walkPath[walkPathIndex]
}

func setWalkDataIndex(index int) {
	if index < 0 || index >= len(walkData) {
		panic("walk frame index out of range")
	}
	walkDataIndex = index
}

func advanceWalkData(count int) {
	setWalkDataIndex(walkDataIndex + count)
}

func currentWalkData() [4]uint16 {
	if walkDataIndex < 0 || walkDataIndex >= len(walkData) {
		panic("walk frame index out of range")
	}
	return walkData[walkDataIndex]
}

func walkDrawItems(frame [4]uint16, behindTree bool) []walkDrawItem {
	items := []walkDrawItem{{
		sprite: frame[3], x: int16(frame[1] - 1), y: int16(frame[2]), flipped: frame[0] != 0,
	}}
	if behindTree {
		// Johnny must be drawn first. The fixed trunk and leaves then cover him
		// while he crosses the path behind the palm tree.
		items = append(items,
			walkDrawItem{sprite: walkTreeTrunkSprite, x: 442, y: 148, tree: true},
			walkDrawItem{sprite: walkTreeLeavesSprite, x: 365, y: 122, tree: true},
		)
	}
	return items
}

func resetWalkDrawOffset() {
	grDx = islandState.xPos
	grDy = islandState.yPos
}

func walkInit(fromSpot, fromHdg, toSpot, toHdg int) {
	walkPath = calcPath(fromSpot, toSpot)
	walkPathIndex = 0

	currentSpot = fromSpot
	currentHdg = fromHdg
	finalSpot = toSpot
	finalHdg = toHdg
	hasArrived = 0
	isBehindTree = 0

	if currentSpot == finalSpot {
		nextSpot = -1
		nextHdg = finalHdg
		lastTurn = 1
	} else {
		nextSpot = nextWalkPathSpot()

		nextHdg = walkDataStartHeadings[currentSpot][nextSpot]
		lastTurn = 0
	}

	increment = (nextHdg - currentHdg) & 0x07
	if increment != 0 {
		if increment < 4 {
			increment = 1
		} else {
			increment = -1
		}
	}
}

func walkAnimate(ttmThread *TTtmThread, ttmBgSlot *TTtmSlot) int {
	ttmSlot := ttmThread.ttmSlot
	sur := ttmThread.ttmLayer
	delay := 0
	// Walking coordinates and the palm-tree occlusion sprites belong to the
	// island. Reset their offset every frame so other TTM drawing cannot make
	// Johnny or the tree jump.
	resetWalkDrawOffset()

	if hasArrived == 0 {

		// Are we turning ?
		if nextHdg != -1 {

			// More than one iteration left? yes, so let's turn
			if (((nextHdg - currentHdg) & 0x07) % 7) > 1 {
				currentHdg = (currentHdg + increment) & 7
				setWalkDataIndex(walkDataBookmarksTurns[currentSpot] + currentHdg)
				if lastTurn != 0 {
					advanceWalkData(9)
				}

				// The turn is over
			} else {

				// Do we have another spot to walk to ?
				if currentSpot != finalSpot {
					nextHdg = -1
					if (currentSpot == 3 && nextSpot == 4) ||
						(currentSpot == 4 && nextSpot == 3) {
						isBehindTree = 1
					} else {
						isBehindTree = 0
					}
					setWalkDataIndex(walkDataBookmarks[currentSpot][nextSpot])
				} else { // Else, we arrived to destination
					setWalkDataIndex(walkDataBookmarksTurns[finalSpot] + finalHdg)
					advanceWalkData(9) // hands in pockets
					hasArrived = 1
				}
			}

			// Walking forward
		} else {

			advanceWalkData(1)

			// Have we reached a spot ? So let's begin a turn...
			if currentWalkData()[1] == 0 {
				currentHdg = walkDataEndHeadings[currentSpot][nextSpot]
				currentSpot = nextSpot

				// What's the next heading ?
				// And the next spot of the path to reach ?
				if currentSpot != finalSpot {
					nextSpot = nextWalkPathSpot()

					nextHdg = walkDataStartHeadings[currentSpot][nextSpot]
				} else {
					nextHdg = finalHdg
					lastTurn = 1
				}

				// Turning: left or right ?
				increment = (nextHdg - currentHdg) & 0x07
				if increment != 0 {
					if increment < 4 {
						increment = 1
					} else {
						increment = -1
					}
				}

				currentHdg = (currentHdg + increment) & 7
				setWalkDataIndex(walkDataBookmarksTurns[currentSpot] + currentHdg)

				if lastTurn != 0 {
					advanceWalkData(9) // hands in pockets
					if currentHdg == finalHdg {
						hasArrived = 1
					}
				}
			}
		}

		frame := currentWalkData()
		debugPrintf("WALKING:  spot=%d hdg=%d next=%d - data %d %d %d %d\n",
			currentSpot, currentHdg, nextHdg,
			frame[0], frame[1], frame[2], frame[3])

		grClearScreen(sur)
		for _, item := range walkDrawItems(frame, isBehindTree != 0) {
			slot := ttmSlot
			if item.tree {
				slot = ttmBgSlot
			}
			if item.flipped {
				grDrawSpriteFlip(sur, slot, item.x, item.y, item.sprite, 0)
			} else {
				grDrawSprite(sur, slot, item.x, item.y, item.sprite, 0)
			}
		}

		if hasArrived != 0 {
			delay = 80
		} else {
			delay = 6
		}
	} else {
		debugPrintln("WALKING: end walk")
		delay = 0
	}

	return delay
}
