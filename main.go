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
)

type App struct {
	db *DB

	output *snd.Output
	input  midi.In
	multi  *notes.NoteMultiplexer
	driver *rtmididrv.Driver

	keys          chan uint8
	pressed       []uint8
	waitingForKey bool

	state          AppState
	stateInSession StateInSessionArgs
}

func (a *App) WaitForKeypress() {
	a.waitingForKey = true
	a.pressed = []uint8{}
	a.keys = make(chan uint8)

	_ = <-a.keys

	a.waitingForKey = false
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

	// Open database
	db, err := DBOpen(viper.Get("DatabasePath").(string))
	if err != nil {
		return nil, err
	}

	// Create app state
	app := App{
		db:     db,
		output: output,
		input:  input,
		multi:  multi,
		driver: driver,
		state:  StateHome}

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
	_ = a.output.Stop()
	a.driver.Close()
	a.input.Close()
}

func (a *App) onNoteOn(p *reader.Position, channel, key, velocity uint8) {
	log.Printf("ON %v (waiting = %v)", key, a.waitingForKey)

	if a.waitingForKey {
		a.pressed = append(a.pressed, key)
		return
	}

	note := notes.MidiToNote(int64(key))
	a.multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, note, float32(velocity)/127))

	switch a.state {
	case StateHome:
		exercises := GetItemsForToday(a.db.Items)

		if len(exercises) == 0 {
			return
		}

		firstExerciseItem := a.db.Items[exercises[0]]
		firstExercise := (ExerciseFromDefinition(firstExerciseItem.Name, firstExerciseItem.ExerciseType, firstExerciseItem.ExerciseDefinition))

		a.state = StateInSession
		a.stateInSession = StateInSessionArgs{
			exercises:       exercises,
			currentIndex:    0,
			currentExercise: &firstExercise,
			state:           ExerciseInProgress,
		}

	case StateInSession:
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

		switch exerciseState {
		case ExerciseFail:
			RenderUI(a)
			a.WaitForKeypress()
			a.stateInSession.currentExercise.Reset()
			a.stateInSession.state = ExerciseInProgress
		case ExercisePass:
			RenderUI(a)
			a.WaitForKeypress()
			a.stateInSession.currentIndex++

			if a.stateInSession.currentIndex == len(a.stateInSession.exercises) {
				a.state = StateHome
			} else {
				nextExerciseItem := a.db.Items[a.stateInSession.exercises[a.stateInSession.currentIndex]]
				nextExercise := ExerciseFromDefinition(nextExerciseItem.Name, nextExerciseItem.ExerciseType, nextExerciseItem.ExerciseDefinition)
				a.stateInSession.state = ExerciseInProgress
				a.stateInSession.currentExercise = &nextExercise
			}
		}
	}

	RenderUI(a)
}

func (a *App) hasKeyBeenPressed(key uint8) bool {
	for _, k := range a.pressed {
		if k == key {
			return true
		}
	}

	return false
}

func (a *App) onNoteOff(p *reader.Position, channel, key, velocity uint8) {
	log.Printf("OFF %v (waiting = %v)", key, a.waitingForKey)

	note := notes.MidiToNote(int64(key))
	a.multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, note, float32(velocity)/127))

	if a.waitingForKey && a.hasKeyBeenPressed(key) {
		a.keys <- key
		return
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
	viper.SetDefault("AKey", "66")
	viper.SetDefault("BKey", "67")
	viper.SetDefault("CKey", "68")
	viper.SetDefault("DKey", "69")
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
