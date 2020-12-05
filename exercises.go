package main

import (
	"gopkg.in/music-theory.v0/note"
)

type ExerciseDefinition struct {
	Name  string
	Parts [][]note.Class
}

var Exercises = []ExerciseDefinition{
	ExerciseDefinition{"C Major", [][]note.Class{
		[]note.Class{note.C},
		[]note.Class{note.D},
		[]note.Class{note.E},
		[]note.Class{note.F},
		[]note.Class{note.G},
		[]note.Class{note.A},
		[]note.Class{note.B},
	}},
	ExerciseDefinition{"C", [][]note.Class{[]note.Class{note.C}}},
	ExerciseDefinition{"C Basic", [][]note.Class{
		[]note.Class{note.C,
			note.E,
			note.G}}},
}

type Exercise struct {
	Definition   ExerciseDefinition
	CurrentStep  int
	CurrentNotes []note.Class
}

func ExerciseFromDefinition(d ExerciseDefinition) Exercise {
	return Exercise{d, 0, []note.Class{}}
}

type ExerciseState uint8

const (
	ExerciseInProgress = iota
	ExerciseFail
	ExercisePass
)

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
		if e.CurrentStep > len(e.Definition.Parts) {
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
