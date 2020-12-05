package main

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gpayer/go-audio-service/generators"
	"github.com/gpayer/go-audio-service/notes"
	"github.com/gpayer/go-audio-service/snd"
	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/reader"
	"gitlab.com/gomidi/rtmididrv"
	mt "gopkg.in/music-theory.v0/note"
	"log"
)

type App struct {
	output *snd.Output
	input  midi.In
	multi  *notes.NoteMultiplexer
	driver *rtmididrv.Driver

	state           ExerciseState
	currentExercise *Exercise
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

	input := ins[1]
	input.Open()

	ex := ExerciseFromDefinition(Exercises[0])

	app := App{
		output:          output,
		input:           input,
		multi:           multi,
		driver:          driver,
		currentExercise: &ex,
		state:           ExerciseInProgress}

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
	note := notes.MidiToNote(int64(key))
	a.multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, note, float32(velocity)/127))

	if a.currentExercise != nil {
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

		a.state = a.currentExercise.Progress(noteClass)

		RenderUI(a)
	}
}

func (a *App) onNoteOff(p *reader.Position, channel, key, velocity uint8) {
	note := notes.MidiToNote(int64(key))
	a.multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, note, float32(velocity)/127))
}

func main() {
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
		if e.Type == ui.KeyboardEvent {
			break
		}
	}
}
