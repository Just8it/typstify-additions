package editor

import (
	"image"
	"testing"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/textstyle/syntax"
)

func newMathDelimiters(t *testing.T, content string, caret int) *mathDelimiters {
	t.Helper()

	state := &gvcode.Editor{}
	state.WithOptions(gvcode.WithColorScheme(syntax.ColorScheme{}))
	state.SetText(content)
	var ops op.Ops
	state.Layout(layout.Context{Ops: &ops, Constraints: layout.Exact(image.Pt(100, 100))}, text.NewShaper())
	state.SetCaret(caret, caret)

	return &mathDelimiters{state: state}
}

// typeText feeds every rune to the input hook, falling back to a plain insertion
// like the default handler does when the hook passes on the input.
func typeText(t *testing.T, md *mathDelimiters, input string) {
	t.Helper()

	for _, r := range input {
		start, end := md.state.Selection()
		if !md.onTextInput(md.state, key.EditEvent{Range: key.Range{Start: start, End: end}, Text: string(r)}) {
			md.state.Insert(string(r))
		}
	}
}

func checkEditor(t *testing.T, md *mathDelimiters, wantText string, wantCaret int) {
	t.Helper()

	if got := md.state.Text(); got != wantText {
		t.Errorf("text = %q, want %q", got, wantText)
	}
	if start, end := md.state.Selection(); start != wantCaret || end != wantCaret {
		t.Errorf("caret = (%d, %d), want (%d, %d)", start, end, wantCaret, wantCaret)
	}
}

func TestTypeMathDelimiters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		caret   int
		input   string
		text    string
		want    int
	}{
		{name: "enters math mode", input: "$", text: "$$", want: 1},
		{name: "pads an empty pair", input: "$ ", text: "$  $", want: 2},
		{name: "keeps typing inside the pair", input: "$x", text: "$x$", want: 2},
		{name: "steps over the closing delimiter", input: "$$", text: "$$", want: 2},
		{name: "steps over a padded closing delimiter", input: "$ $", text: "$  $", want: 4},
		{name: "steps over the closing delimiter after content", input: "$x$", text: "$x$", want: 3},
		{name: "pairs inside a sentence", content: "let  hold", caret: 4, input: "$", text: "let $$ hold", want: 5},
		{name: "skips pairing after a word", content: "x", caret: 1, input: "$", text: "x$", want: 2},
		{name: "skips pairing before a word", content: "x", caret: 0, input: "$", text: "$x", want: 1},
		{name: "pads only an empty pair", content: "$x$", caret: 2, input: " ", text: "$x $", want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := newMathDelimiters(t, tc.content, tc.caret)
			typeText(t, md, tc.input)
			checkEditor(t, md, tc.text, tc.want)
		})
	}
}

func TestDeleteMathDelimiters(t *testing.T) {
	for _, tc := range []struct {
		name       string
		input      string
		backspaces int
		text       string
		want       int
	}{
		{name: "deletes both halves of an empty pair", input: "$", backspaces: 1, text: "", want: 0},
		{name: "deletes the padding before the pair", input: "$ ", backspaces: 1, text: "$ $", want: 1},
		{name: "deletes both halves of a padded pair", input: "$ ", backspaces: 2, text: " ", want: 0},
		{name: "deletes content before the pair", input: "$x", backspaces: 1, text: "$$", want: 1},
		{name: "deletes both halves once the content is gone", input: "$x", backspaces: 2, text: "", want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md := newMathDelimiters(t, "", 0)
			typeText(t, md, tc.input)
			for range tc.backspaces {
				if evt := md.onDeleteBackward(layout.Context{}, key.Event{}); evt == nil {
					t.Fatal("expected a change event from backspace")
				}
			}
			checkEditor(t, md, tc.text, tc.want)
		})
	}
}

func TestDeleteMathDelimitersKeepsNeighbours(t *testing.T) {
	md := newMathDelimiters(t, "$a$", 3)
	typeText(t, md, " $")
	checkEditor(t, md, "$a$ $$", 5)

	if evt := md.onDeleteBackward(layout.Context{}, key.Event{}); evt == nil {
		t.Fatal("expected a change event from backspace")
	}
	checkEditor(t, md, "$a$ ", 4)
}

func TestDeleteBackwardOutsideMathPair(t *testing.T) {
	md := newMathDelimiters(t, "note", 4)

	if evt := md.onDeleteBackward(layout.Context{}, key.Event{}); evt == nil {
		t.Fatal("expected a change event from backspace")
	}
	checkEditor(t, md, "not", 3)
}

func TestTypstLanguageExtension(t *testing.T) {
	if len(langExtRegistry.optionsFor("notes.typ")) == 0 {
		t.Error("expected the Typst extension to be registered for .typ files")
	}
	if opts := langExtRegistry.optionsFor("notes.md"); len(opts) != 0 {
		t.Errorf("got %d options for .md files, want none", len(opts))
	}
}
