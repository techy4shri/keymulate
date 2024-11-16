package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto"
	"github.com/halfloafhq/keymulate/internal/kbd"
	hook "github.com/robotn/gohook"
)

func main() {
	switchPtr := flag.String("switch", "cream", "The sound of switches to output")
	flag.Parse()

	if _, err := os.Stat(fmt.Sprintf("audio/%s", *switchPtr)); os.IsNotExist(err) {
		log.Fatalf("Keyboard sounds not found: %s", *switchPtr)
	}

	s := &State{
		sounds: map[string][]byte{},
		keyMap: map[uint16]bool{},
	}

	s.loadSoundsForKeyboard(*switchPtr)
	// Create an Oto context (for audio playback)
	context, err := oto.NewContext(48000, 2, 2, 8192)
	if err != nil {
		log.Fatalf("failed to create Oto context: %v", err)
	}
	defer context.Close()

	// Function to play sound in a goroutine
	playSound := func(key string) {
		// Create an MP3 decoder
		decoder, err := mp3.NewDecoder(bytes.NewReader(s.sounds[key]))
		if err != nil {
			log.Fatalf("failed to create MP3 decoder: %v", err)
		}

		player := context.NewPlayer()
		defer player.Close()

		// Reset the decoder (so it plays from the beginning)
		decoder.Seek(0, 0)

		// Create a buffer to read the decoded audio
		buf := make([]byte, 8192)
		for {
			n, err := decoder.Read(buf)
			if err != nil && err != io.EOF {
				log.Printf("failed to read decoded audio: %v", err)
				break
			}
			if n == 0 {
				break
			}
			player.Write(buf[:n])
		}
	}

	var wg sync.WaitGroup

	scheduleSound := func(key string) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			playSound(key)
		}()
	}

	hook.Register(hook.KeyDown, []string{"A-Z a-z 0-9"}, func(e hook.Event) {

		if s.keyMap[e.Rawcode] {
			return
		}
		s.keyMap[e.Rawcode] = true

		if e.Rawcode == kbd.ENTER {
			scheduleSound(kbd.PRESS_ENTER)
		} else if e.Rawcode == kbd.SPACE {
			scheduleSound(kbd.PRESS_SPACE)
		} else {
			scheduleSound(getPressKey())
		}
	})

	hook.Register(hook.KeyUp, []string{"A-Z a-z 0-9"}, func(e hook.Event) {
		if !s.keyMap[e.Rawcode] {
			return
		}
		s.keyMap[e.Rawcode] = false

		if e.Rawcode == kbd.ENTER {
			scheduleSound(kbd.RELEASE_ENTER)
		} else if e.Rawcode == kbd.SPACE {
			scheduleSound(kbd.RELEASE_SPACE)
		} else {
			scheduleSound(kbd.RELEASE_GENERIC)
		}
	})

	hs := hook.Start()
	<-hook.Process(hs)
	wg.Wait()

}

func getPressKey() string {
	keys := []string{kbd.PRESS_GENERIC_R0, kbd.PRESS_GENERIC_R1, kbd.PRESS_GENERIC_R2, kbd.PRESS_GENERIC_R3, kbd.PRESS_GENERIC_R4}
	key := keys[rand.Intn(len(keys))]
	return key
}

type State struct {
	sounds map[string][]byte
	keyMap map[uint16]bool
}

func (s *State) loadSound(kbdSwitch string, soundName string) {
	soundFile, err := os.Open(fmt.Sprintf("./audio/%s/%s.mp3", kbdSwitch, soundName))

	if err != nil {
		log.Fatalf("failed to open sound file: %v", err)
	}
	defer soundFile.Close()

	sound, err := io.ReadAll(soundFile)
	if err != nil {
		log.Fatalf("failed to read sound file: %v", err)
	}

	s.sounds[soundName] = sound
}

func (s *State) loadSoundsForKeyboard(kbdSwitch string) {
	keys := []string{kbd.PRESS_BACKSPACE, kbd.PRESS_ENTER, kbd.PRESS_GENERIC_R0, kbd.PRESS_GENERIC_R1, kbd.PRESS_GENERIC_R2, kbd.PRESS_GENERIC_R3, kbd.PRESS_GENERIC_R4, kbd.PRESS_SPACE, kbd.RELEASE_BACKSPACE, kbd.RELEASE_ENTER, kbd.RELEASE_GENERIC, kbd.RELEASE_SPACE}
	for _, key := range keys {
		s.loadSound(kbdSwitch, key)
	}
}
