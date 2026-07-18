package main

import (
	"fmt"
	"log"
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// r.c. This lame "sound engine" was fundamentally different and way simpler than the C original.

var (
	sfx = []string{
		"sound0.wav",
		"sound1.wav",
		"sound2.wav",
		"sound3.wav",
		"sound4.wav",
		"sound5.wav",
		"sound6.wav",
		"sound7.wav",
		"sound8.wav",
		"sound9.wav",
		"sound10.wav",
		"missing",
		"sound12.wav",
		"missing",
		"sound14.wav",
		"sound15.wav",
		"sound16.wav",
		"sound17.wav",
		"sound18.wav",
		"sound19.wav",
		"sound20.wav",
		"sound21.wav",
		"sound22.wav",
		"sound23.wav",
		"sound24.wav",
	}
	soundSfx = make([]rl.Sound, len(sfx))
)

func loadSfx() error {
	for i, filename := range sfx {
		if filename == "missing" {
			continue
		}
		soundPath := filepath.Join(appSettings.dataDir, filename)
		snd := rl.LoadSound(soundPath)
		if !rl.IsSoundValid(snd) {
			log.Printf("optional sound %d (%s) was not found or could not be decoded", i, soundPath)
			continue
		}
		rl.SetSoundVolume(snd, 0.5)
		soundSfx[i] = snd
	}
	return nil
}

func unloadSfx() {
	for i, snd := range soundSfx {
		if sfx[i] == "missing" || !rl.IsSoundValid(snd) {
			continue
		}
		rl.UnloadSound(snd)
	}
}

func soundStopAll() {
	if !audioReady {
		return
	}
	for i, snd := range soundSfx {
		if sfx[i] != "missing" && rl.IsSoundValid(snd) {
			rl.StopSound(snd)
		}
	}
}

func soundSetPaused(paused bool) {
	if !audioReady {
		return
	}
	for i, snd := range soundSfx {
		if sfx[i] == "missing" || !rl.IsSoundValid(snd) {
			continue
		}
		if paused {
			rl.PauseSound(snd)
		} else {
			rl.ResumeSound(snd)
		}
	}
}

func soundPlay(id uint16) {
	if !audioReady || appSettings.mute {
		return
	}
	if int(id) > len(soundSfx)-1 {
		fmt.Printf("sound id index out of range:%d\n", id)
		return
	}
	if sfx[id] == "missing" {
		fmt.Println("missing audio for this id =>", id)
		return
	}
	snd := soundSfx[id]
	if !rl.IsSoundValid(snd) {
		log.Printf("sound id %d is not valid", id)
		return
	}
	if appSettings.ttm != "" {
		log.Printf("playing sound id=%d file=%s", id, sfx[id])
	}
	rl.PlaySound(snd)
	if playbackPaused {
		rl.PauseSound(snd)
	}
}
