package main

type AppState uint8

// Application is modelled as a simple state machine
const (
	StateHome = iota
	StateInSession
)

type ExerciseState uint8

const (
	ExerciseInProgress = iota
	ExerciseFail
	ExercisePass
)

type StateHomeArgs struct{}

type StateInSessionArgs struct {
	cards           []Card
	currentIndex    int
	currentExercise *Exercise
	state           ExerciseState
	showHint        bool
}
