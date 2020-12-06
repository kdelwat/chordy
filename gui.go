package main

import (
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	mt "gopkg.in/music-theory.v0/note"
	"image"
)

var NormalStyle ui.Style = ui.NewStyle(ui.ColorWhite)
var CurrentStyle ui.Style = ui.NewStyle(ui.ColorYellow)
var SuccessStyle ui.Style = ui.NewStyle(ui.ColorGreen)
var FailStyle ui.Style = ui.NewStyle(ui.ColorRed)

type ExerciseWidget struct {
	ui.Block
	e     *Exercise
	card  *Card
	state ExerciseState
}

func NewExerciseWidget(e *Exercise, s ExerciseState, card *Card) *ExerciseWidget {
	return &ExerciseWidget{Block: *ui.NewBlock(), e: e, state: s, card: card}
}

func (self *ExerciseWidget) DrawText(buf *ui.Buffer, text string, x, y int, style ui.Style) {
	for i, c := range text {
		buf.SetCell(ui.NewCell(c, style), image.Pt(x+i, y))
	}
}

func (self *ExerciseWidget) Draw(buf *ui.Buffer) {
	self.Block.Draw(buf)

	// Draw info
	self.DrawText(buf, fmt.Sprintf("Exercise: %s", self.e.Definition.Name),
		self.Inner.Min.X,
		self.Inner.Min.Y,
		NormalStyle)

	if self.state == ExerciseFail {
		self.DrawText(buf, "FAILED. Make any selection to continue.",
			self.Inner.Min.X,
			self.Inner.Min.Y+1,
			FailStyle)
	} else if self.state == ExercisePass {
		self.DrawText(buf, "PASSED. Make any selection to continue.",
			self.Inner.Min.X,
			self.Inner.Min.Y+1,
			SuccessStyle)
	}

	// Draw progress boxes
	width := len(self.e.Definition.Parts)
	startX := self.Inner.Min.X - 1 + ((self.Inner.Max.X-self.Inner.Min.X)-width)/2
	startY := self.Inner.Min.Y + ((self.Inner.Max.Y - self.Inner.Min.Y) / 2)

	for i := 0; i < width; i++ {
		var icon rune
		var style ui.Style

		if self.state == ExerciseFail && self.e.CurrentStep == i {
			icon = '▣'
			style = FailStyle
		} else if self.e.CurrentStep > i {
			icon = '▣'
			style = SuccessStyle
		} else if self.e.CurrentStep == i {
			icon = '□'
			style = CurrentStyle
		} else {
			icon = '□'
			style = NormalStyle
		}
		buf.SetCell(ui.NewCell(icon, style), image.Pt(startX+(i*2), startY))
		buf.SetCell(ui.NewCell(' '), image.Pt(startX+(i*2)+1, startY))

		if self.state == ExerciseFail {
			y := 1
			for _, note := range self.e.Definition.Parts[i] {
				adj := mt.AdjSymbolOf(self.card.ExerciseDefinition)
				for _, c := range note.String(adj) {
					buf.SetCell(ui.NewCell(c, FailStyle), image.Pt(startX+(i*2), startY+y+1))
					y++
				}
				y++
			}
		}
	}
}

// MAIN UI

func InitUI() error {
	if err := ui.Init(); err != nil {
		return err
	}

	return nil
}

func CloseUI() {
	ui.Close()
}

func ClearUI() {
	ui.Clear()
}

func RenderUI(app *App) {
	switch app.state {
	case StateHome:
		renderHome(app)
	case StateInSession:
		renderInSession(app)
	}
}

func padRow(ratio float64, a, b, c, d string) ui.GridItem {
	aP := widgets.NewParagraph()
	aP.Text = a
	bP := widgets.NewParagraph()
	bP.Text = b
	cP := widgets.NewParagraph()
	cP.Text = c
	dP := widgets.NewParagraph()
	dP.Text = d

	return ui.NewRow(
		ratio,
		ui.NewCol(1.0/4, aP),
		ui.NewCol(1.0/4, bP),
		ui.NewCol(1.0/4, cP),
		ui.NewCol(1.0/4, dP),
	)
}

func renderHome(app *App) {
	p := widgets.NewParagraph()
	p.Text = "Chordy"
	p.SetRect(0, 0, 25, 5)

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0/2, ui.NewCol(1.0, p)),
	)

	ui.Render(grid)
}

func renderInSession(app *App) {
	p := widgets.NewGauge()
	p.Title = "Session Progress"
	p.Percent = int(100.0 * float32(app.stateInSession.currentIndex) / float32(len(app.stateInSession.cards)))
	p.Label = fmt.Sprintf("%v%% (%v/%v)", p.Percent, app.stateInSession.currentIndex, len(app.stateInSession.cards))

	info := widgets.NewParagraph()
	info.Title = "Current Exercise"
	card := app.stateInSession.cards[app.stateInSession.currentIndex]
	var lastSeen string
	if card.LastRecalledAt.IsZero() {
		lastSeen = "never"
	} else {
		lastSeen = fmt.Sprintf("%v", card.LastRecalledAt)
	}

	info.Text = fmt.Sprintf("Name: %v\nLast seen: %v\nEstimated difficulty: %v", card.Name, lastSeen, card.Ef)

	e := NewExerciseWidget(app.stateInSession.currentExercise, app.stateInSession.state, &app.stateInSession.cards[app.stateInSession.currentIndex])

	var pads ui.GridItem

	switch app.stateInSession.state {
	case ExerciseInProgress:
		pads = padRow(1.0/4, "Give up", "Hint", "", "")
	case ExerciseFail:
		pads = padRow(1.0/4, "Retry", "Continue", "", "")
	case ExercisePass:
		pads = padRow(1.0/4, "Easy", "Normal", "Hard", "")
	}

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0/4, ui.NewCol(1.0/2, info), ui.NewCol(1.0/2, p)),
		ui.NewRow(3*1.0/4-(1.0/8), ui.NewCol(1.0, e)),
		pads,
	)

	ui.Render(grid)
}
