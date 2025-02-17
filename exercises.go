package main

import (
	"gopkg.in/music-theory.v0/chord"
	"gopkg.in/music-theory.v0/note"
	"gopkg.in/music-theory.v0/scale"
)

type ExerciseDefinition struct {
	Name  string
	Parts [][]note.Class
}

type Exercise struct {
	Definition   ExerciseDefinition
	CurrentStep  int
	CurrentNotes []note.Class
}

func parseNote(n string) note.Class {
	switch n {
	case "C":
		return note.C
	case "C#":
		return note.Cs
	case "Cb":
		return note.Cs
	case "D":
		return note.D
	case "D#":
		return note.Ds
	case "Db":
		return note.Ds
	case "E":
		return note.E
	case "F":
		return note.F
	case "F#":
		return note.Fs
	case "Fb":
		return note.Fs
	case "G":
		return note.G
	case "G#":
		return note.Gs
	case "Gb":
		return note.Gs
	case "A":
		return note.A
	case "A#":
		return note.As
	case "Ab":
		return note.As
	case "B":
		return note.B
	}

	return note.Nil
}

func notesToClasses(notes []*note.Note) []note.Class {
	classes := []note.Class{}

	for _, n := range notes {
		classes = append(classes, n.Class)
	}

	return classes
}

func CreateExercise(card Card) Exercise {
	var definition ExerciseDefinition
	definition.Name = card.Name

	switch card.ExerciseType {
	case "note":
		definition.Parts = [][]note.Class{[]note.Class{parseNote(card.ExerciseDefinition)}}
	case "chord":
		c := chord.Of(card.ExerciseDefinition)
		definition.Parts = [][]note.Class{notesToClasses((&c).Notes())}
	case "scale":
		s := scale.Of(card.ExerciseDefinition)
		definition.Parts = [][]note.Class{}
		for _, n := range notesToClasses((&s).Notes()) {
			definition.Parts = append(definition.Parts, []note.Class{n})
		}
	}

	return Exercise{
		Definition:   definition,
		CurrentStep:  0,
		CurrentNotes: []note.Class{},
	}
}

func (e *Exercise) Reset() {
	e.CurrentStep = 0
	e.CurrentNotes = []note.Class{}
}

func (e *Exercise) Progress(n note.Class) ExerciseState {
	// Ignore repeated note presses
	if noteArrayContains(e.CurrentNotes, n) {
		return ExerciseInProgress
	}

	// Fail if incorrect note played
	if !noteArrayContains(e.Definition.Parts[e.CurrentStep], n) {
		return ExerciseFail
	}

	// Otherwise, note is correct and should be added to current notes
	e.CurrentNotes = append(e.CurrentNotes, n)

	// If this step is complete, go to the next step or return success
	if len(e.CurrentNotes) == len(e.Definition.Parts[e.CurrentStep]) {
		e.CurrentStep = e.CurrentStep + 1
		e.CurrentNotes = []note.Class{}
		if e.CurrentStep >= len(e.Definition.Parts) {
			return ExercisePass
		}
	}

	return ExerciseInProgress
}

func noteArrayContains(notes []note.Class, n note.Class) bool {
	for _, x := range notes {
		if x == n {
			return true
		}
	}

	return false
}
