package main

import (
	"errors"
	ui "github.com/gizak/termui/v3"
	"github.com/gpayer/go-audio-service/generators"
	"github.com/gpayer/go-audio-service/notes"
	"github.com/gpayer/go-audio-service/snd"
	"github.com/spf13/viper"
	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/reader"
	"gitlab.com/gomidi/rtmididrv"
	mt "gopkg.in/music-theory.v0/note"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type SelectionKey uint8

const (
	KeyInvalid = iota
	KeyA
	KeyB
	KeyC
	KeyD
)

type MidiResources struct {
	output *snd.Output
	input  midi.In
	multi  *notes.NoteMultiplexer
	driver *rtmididrv.Driver
}

type SelectionState struct {
	waiting             bool
	pressedWhileWaiting []uint8
}

type App struct {
	db *DB

	midi MidiResources

	selection SelectionState

	state          AppState
	stateInSession StateInSessionArgs
}

func (a *App) WaitForSelection() {
	a.selection.waiting = true
	a.selection.pressedWhileWaiting = []uint8{}
}

func (a *App) SelectionReady(key uint8) bool {
	for _, k := range a.selection.pressedWhileWaiting {
		if k == key {
			a.selection.waiting = false
			return true
		}
	}

	return false
}

func getSelectionKey(key uint8) SelectionKey {
	keyString := strconv.Itoa(int(key))

	switch keyString {
	case viper.Get("AKey"):
		return KeyA
	case viper.Get("BKey"):
		return KeyB
	case viper.Get("CKey"):
		return KeyC
	case viper.Get("DKey"):
		return KeyD
	}

	return KeyInvalid
}

func isSelectionKey(key uint8) bool {
	selectionKey := getSelectionKey(key)

	return selectionKey != KeyInvalid
}

func InitApp() (*App, error) {
	// Set up output stream
	output, err := snd.NewOutput(44000, 512)
	if err != nil {
		return nil, err
	}

	rect := generators.NewRect(44000, 440.0)
	multi := notes.NewNoteMultiplexer()
	multi.SetReadable(rect)
	output.SetReadable(multi)

	err = output.Start()
	if err != nil {
		return nil, err
	}

	// Set up input stream
	driver, err := rtmididrv.New()
	if err != nil {
		return nil, err
	}

	ins, err := driver.Ins()
	if err != nil {
		return nil, err
	}

	if len(ins) < 2 {
		return nil, errors.New("no MIDI input device found")
	}

	input := ins[1]
	input.Open()

	midi := MidiResources{
		output: output,
		input:  input,
		multi:  multi,
		driver: driver,
	}

	// Open database
	db, err := Connect(viper.Get("DatabasePath").(string))
	if err != nil {
		return nil, err
	}

	// Create app state
	app := App{
		db:        db,
		midi:      midi,
		selection: SelectionState{},
		state:     StateHome,
	}

	// Set up MIDI event handlers
	rd := reader.New(
		reader.NoLogger(),
		reader.NoteOn(app.onNoteOn),
		reader.NoteOff(app.onNoteOff),
	)

	err = rd.ListenTo(input)
	if err != nil {
		return nil, err
	}

	return &app, nil
}

func (a *App) Stop() {
	_ = a.midi.output.Stop()
	a.midi.driver.Close()
	a.midi.input.Close()
	a.db.Close()
}

// Handle MIDI NOTEON events
func (a *App) onNoteOn(p *reader.Position, channel, key, velocity uint8) {
	// If waiting for a selection (pad press), store the pressed key
	// and return - need to wait for the NOTEOFF before proceeding
	if a.selection.waiting {
		if isSelectionKey(key) {
			a.selection.pressedWhileWaiting = append(a.selection.pressedWhileWaiting, key)
		}
		return
	}

	// Play the pressed note
	note := notes.MidiToNote(int64(key))
	a.midi.multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, note, float32(velocity)/127))

	// Process event according to the current state
	switch a.state {
	case StateHome:
		cardsForThisSession, err := a.db.GetCardsForToday()

		if err != nil {
			panic(err) // This should never happen
		}

		if len(cardsForThisSession) == 0 {
			return
		}

		currentExercise := CreateExercise(cardsForThisSession[0])

		a.state = StateInSession
		a.stateInSession = StateInSessionArgs{
			cards:           cardsForThisSession,
			currentIndex:    0,
			currentExercise: &currentExercise,
			state:           ExerciseInProgress,
		}

	case StateInSession:
		// Check pads first
		if getSelectionKey(key) == KeyA {
			a.stateInSession.state = ExerciseFail
		} else if getSelectionKey(key) == KeyB {
			a.stateInSession.showHint = true
		} else {
			var noteClass mt.Class
			switch key % 12 {
			case 0:
				noteClass = mt.C
			case 1:
				noteClass = mt.Cs
			case 2:
				noteClass = mt.D
			case 3:
				noteClass = mt.Ds
			case 4:
				noteClass = mt.E
			case 5:
				noteClass = mt.F
			case 6:
				noteClass = mt.Fs
			case 7:
				noteClass = mt.G
			case 8:
				noteClass = mt.Gs
			case 9:
				noteClass = mt.A
			case 10:
				noteClass = mt.As
			case 11:
				noteClass = mt.B
			}

			exerciseState := a.stateInSession.currentExercise.Progress(noteClass)
			a.stateInSession.state = exerciseState
		}

		switch a.stateInSession.state {
		case ExerciseFail:
			a.WaitForSelection()
		case ExercisePass:
			a.WaitForSelection()
		}
	}

	RenderUI(a)
}

func (a *App) onNoteOff(p *reader.Position, channel, key, velocity uint8) {
	note := notes.MidiToNote(int64(key))
	a.midi.multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, note, float32(velocity)/127))

	switch a.state {
	case StateInSession:
		switch a.stateInSession.state {
		case ExerciseFail:
			if a.SelectionReady(key) {
				if getSelectionKey(key) == KeyA {
					a.stateInSession.currentExercise.Reset()
					a.stateInSession.state = ExerciseInProgress
				} else {
					updatedCard := RecalculateCard(a.stateInSession.cards[a.stateInSession.currentIndex], 0)
					a.db.Upsert(updatedCard)

					a.stateInSession.currentIndex++

					if a.stateInSession.currentIndex == len(a.stateInSession.cards) {
						a.state = StateHome
					} else {
						currentExercise := CreateExercise(a.stateInSession.cards[a.stateInSession.currentIndex])
						a.stateInSession.state = ExerciseInProgress
						a.stateInSession.currentExercise = &currentExercise
						a.stateInSession.showHint = false
					}
				}

			}
		case ExercisePass:
			// Wait for the user to make a pad selection
			if a.SelectionReady(key) {
				// Based on the pad pressed, determine the difficulty level of the exercise
				selectionKey := getSelectionKey(key)
				var difficulty uint
				switch selectionKey {
				case KeyA:
					difficulty = 3
				case KeyB:
					difficulty = 4
				case KeyC:
					difficulty = 5
				case KeyD:
					difficulty = 5
				}

				updatedCard := RecalculateCard(a.stateInSession.cards[a.stateInSession.currentIndex], difficulty)

				a.db.Upsert(updatedCard)

				a.stateInSession.currentIndex++

				if a.stateInSession.currentIndex == len(a.stateInSession.cards) {
					a.state = StateHome
				} else {
					currentExercise := CreateExercise(a.stateInSession.cards[a.stateInSession.currentIndex])
					a.stateInSession.state = ExerciseInProgress
					a.stateInSession.currentExercise = &currentExercise
					a.stateInSession.showHint = false
				}
			}
		}
	}

	RenderUI(a)
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("could not find user home directory: %v", err)
	}

	configPath := filepath.Join(home, "/.config/chordy")
	defaultDataPath := filepath.Join(home, "/.data/chordy")

	if err = os.MkdirAll(configPath, os.ModePerm); err != nil {
		log.Fatalf("could not create config directory: %v", err)
	}

	if err = os.MkdirAll(defaultDataPath, os.ModePerm); err != nil {
		log.Fatalf("could not create config directory: %v", err)
	}

	viper.SetDefault("DatabasePath", filepath.Join(defaultDataPath, "db.json"))
	viper.SetDefault("AKey", "40")
	viper.SetDefault("BKey", "41")
	viper.SetDefault("CKey", "42")
	viper.SetDefault("DKey", "43")
	viper.SetConfigName("config.json")
	viper.AddConfigPath(configPath)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {

			if err = viper.WriteConfigAs(filepath.Join(configPath, "config.json")); err != nil {
				log.Fatalf("could not write default config file: %v", err)
			}
		} else {
			log.Fatalf("could not read config file: %v", err)
		}
	}

	app, err := InitApp()

	if err != nil {
		log.Fatalf("could not initialize app: %v", err)
	}

	defer app.Stop()

	if err := InitUI(); err != nil {
		log.Fatalf("could not initialize ui: %v", err)
	}

	defer CloseUI()

	RenderUI(app)

	for e := range ui.PollEvents() {
		switch e.ID {
		case "q", "<C-c>":
			return
		case "<Resize>":
			ClearUI()
			RenderUI(app)
		}
	}
}
