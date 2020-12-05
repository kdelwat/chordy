package main

import (
	"gopkg.in/music-theory.v0/note"
)

type Exercise struct {
	Name string,
	Parts [][]note.Class
}

var Exercises = []Exercise{
	Exercise{"C", [][]note.Class{[]note.Class{note.C}}}
}

