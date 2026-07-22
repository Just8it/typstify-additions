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

func TestInsertTypstLineBreak(t *testing.T) {
	state := &gvcode.Editor{}
	state.WithOptions(gvcode.WithColorScheme(syntax.ColorScheme{}))
	state.SetText("  Text")
	var ops op.Ops
	state.Layout(layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(100, 100))}, text.NewShaper())
	state.SetCaret(state.Len(), state.Len())
	editor := TextEditor{state: state}

	if !editor.insertTypstLineBreak() {
		t.Fatal("expected Typst line break")
	}
	if got := state.Text(); got != "  Text \\\n  " {
		t.Fatalf("got %q, want a forced line break with preserved indentation", got)
	}
}
