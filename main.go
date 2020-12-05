package main

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/gpayer/go-audio-service/generators"
	"github.com/gpayer/go-audio-service/notes"
	"github.com/gpayer/go-audio-service/snd"
	"gitlab.com/gomidi/midi"
	"gitlab.com/gomidi/midi/reader"
	"gitlab.com/gomidi/rtmididrv"
	"log"
)

type App struct {
	output *snd.Output
	input  midi.In
	multi  *notes.NoteMultiplexer
	driver *rtmididrv.Driver
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

	app := App{output, input, multi, driver}

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

	if err := ui.Init(); err != nil {
		log.Fatalf("could not initialize UI: %v", err)
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	p.Text = "Chordy"
	p.SetRect(0, 0, 25, 5)

	ui.Render(p)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
	}
}
