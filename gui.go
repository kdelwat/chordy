package main

import (
	"fmt"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"image"
)

var NormalStyle ui.Style = ui.NewStyle(ui.ColorWhite)
var CurrentStyle ui.Style = ui.NewStyle(ui.ColorYellow)
var SuccessStyle ui.Style = ui.NewStyle(ui.ColorGreen)
var FailStyle ui.Style = ui.NewStyle(ui.ColorRed)

type ExerciseWidget struct {
	ui.Block
	e     *Exercise
	state ExerciseState
}

func NewExerciseWidget(e *Exercise, s ExerciseState) *ExerciseWidget {
	return &ExerciseWidget{Block: *ui.NewBlock(), e: e, state: s}
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
		self.DrawText(buf, "FAILED. Play any note to continue.",
			self.Inner.Min.X,
			self.Inner.Min.Y+1,
			FailStyle)
	} else if self.state == ExercisePass {
		self.DrawText(buf, "PASSED. Play any note to continue.",
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
	p := widgets.NewParagraph()
	p.Text = "Chordy"
	p.SetRect(0, 0, 25, 5)

	e := NewExerciseWidget(app.currentExercise, app.state)
	e.SetRect(0, 6, 25, 11)

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0/2, ui.NewCol(1.0, p)),
		ui.NewRow(1.0/2, ui.NewCol(1.0, e)),
	)

	ui.Render(grid)
}
