package main

import "github.com/rakyll/portmidi"
import "log"
import "fmt"
import "github.com/gpayer/go-audio-service/generators"
import "github.com/gpayer/go-audio-service/notes"
import "github.com/gpayer/go-audio-service/snd"
import "time"

func onMidiEvent(multi *notes.NoteMultiplexer, e portmidi.Event) {
	log.Printf("[MIDI] %x %x\n", e.Status, e.Data1, e.Data2)
	switch e.Status {
	case 0x90:
		// NOTE ON
		note := notes.MidiToNote(e.Data1)
		multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, note, 0.1))

	case 0x80:
		// NOTE OFF
		note := notes.MidiToNote(e.Data1)
		multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, note, 0.1))
	}
}

func main() {
	fmt.Println("Welcome to Chordy")

	portmidi.Initialize()
	defer portmidi.Terminate()

	fmt.Println("Number of MIDI devices: ", portmidi.CountDevices())
	fmt.Println("Default input device: ", portmidi.DefaultInputDeviceID())

	in, err := portmidi.NewInputStream(3, 1024)

	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	output, err := snd.NewOutput(44000, 512)
	if err != nil {
		panic(err)
	}
	defer output.Close()

	rect := generators.NewRect(44000, 440.0)
	multi := notes.NewNoteMultiplexer()
	multi.SetReadable(rect)
	output.SetReadable(multi)

	err = output.Start()
	if err != nil {
		panic(err)
	}

	ch := in.Listen()

	for event := range ch {
		onMidiEvent(multi, event)
	}

	multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, notes.Note(notes.C, 3), 0.1))
	time.Sleep(500 * time.Millisecond)
	multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, notes.Note(notes.E, 3), 0.1))
	time.Sleep(500 * time.Millisecond)
	multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, notes.Note(notes.G, 3), 0.1))
	time.Sleep(750 * time.Millisecond)
	multi.SendNoteEvent(notes.NewNoteEvent(notes.Pressed, notes.Note(notes.G, 2), 0.1))
	time.Sleep(1000 * time.Millisecond)
	multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, notes.Note(notes.C, 3), 0.0))
	multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, notes.Note(notes.E, 3), 0.0))
	multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, notes.Note(notes.G, 3), 0.0))
	time.Sleep(5000 * time.Millisecond)
	multi.SendNoteEvent(notes.NewNoteEvent(notes.Released, notes.Note(notes.G, 2), 0.0))

	_ = output.Stop()
}
