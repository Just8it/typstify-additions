package editor

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/textstyle/syntax"
)

func TestInsertMathPairSpace(t *testing.T) {
	if !shouldInsertMathPairSpace("notes.typ", "$ $") {
		t.Fatal("expected math pair spacing")
	}
}

func TestDeleteTrackedMathPair(t *testing.T) {
	state := &gvcode.Editor{}
	state.WithOptions(gvcode.WithColorScheme(syntax.ColorScheme{}))
	state.SetText(" $")
	ed := TextEditor{
		state:    state,
		mathPair: &trackedMathPair{start: 0, end: 2, textLen: 3},
	}
	var ops op.Ops
	state.Layout(layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(100, 100))}, text.NewShaper())
	if !ed.updateMathPair() {
		t.Fatal("expected tracked closing dollar deletion")
	}

	if got := state.Text(); got != " " {
		t.Fatalf("got %q, want one untouched space", got)
	}
}
