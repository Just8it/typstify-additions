package editor

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/oligo/gvcode"
)

const mathDelim = '$'

func init() {
	RegisterLanguageExtension(LanguageExtension{
		Lang: "Typst",
		Exts: []string{".typ"},
		Opts: []TextEditorOption{withMathDelimiters()},
	})
}

// withMathDelimiters pairs the dollar signs enclosing math mode, the way brackets
// and quotes are paired. The option runs once per opened file, so the tracked
// pair stays local to a single editor.
func withMathDelimiters() TextEditorOption {
	return func(ed *TextEditor) {
		md := &mathDelimiters{state: ed.state}
		WithTextInputHook(md.onTextInput)(ed)
		// Deletion has no hook to attach to. Alt/Ctrl+Backspace is left out of the
		// filter so it keeps falling through to the builtin word deletion.
		ed.state.RegisterCommand(md, key.Filter{Name: key.NameDeleteBackward, Optional: key.ModShift}, md.onDeleteBackward)
	}
}

// mathDelimiters auto-pairs Typst math delimiters, tracking the generated pair by
// markers so both halves stay addressable while the text around them changes.
type mathDelimiters struct {
	state            *gvcode.Editor
	opening, closing *gvcode.Marker
}

func (md *mathDelimiters) onTextInput(ed *gvcode.Editor, ke key.EditEvent) bool {
	switch ke.Text {
	case string(mathDelim):
		return md.insertPair(ke)
	case " ":
		return md.padPair(ke)
	default:
		return false
	}
}

func (md *mathDelimiters) insertPair(ke key.EditEvent) bool {
	if ke.Range.Start != ke.Range.End {
		return false
	}
	caret := ke.Range.Start

	if opening, closing, ok := md.pair(); ok && caret > opening && caret <= closing &&
		strings.TrimSpace(md.state.ReadTextBetween(caret, closing)) == "" {
		md.state.SetCaret(closing+1, closing+1)
		md.forget()
		return true
	}

	// Next to a word the dollar sign is more likely to close a math block that
	// was opened by hand.
	if isWordRune(md.runeAt(caret-1)) || isWordRune(md.runeAt(caret)) {
		return false
	}

	md.state.SetCaret(caret, caret)
	md.state.Insert(string([]rune{mathDelim, mathDelim}))
	md.state.SetCaret(caret+1, caret+1)
	md.track(caret, caret+1)
	return true
}

// padPair turns "$|$" into "$ | $", as Typst renders inline math spaced out.
func (md *mathDelimiters) padPair(ke key.EditEvent) bool {
	if ke.Range.Start != ke.Range.End {
		return false
	}
	caret := ke.Range.Start
	if md.runeAt(caret-1) != mathDelim || md.runeAt(caret) != mathDelim {
		return false
	}

	md.state.SetCaret(caret, caret)
	md.state.Insert("  ")
	md.state.SetCaret(caret+1, caret+1)
	md.track(caret-1, caret+2)
	return true
}

func (md *mathDelimiters) onDeleteBackward(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
	if md.state.Mode() == gvcode.ModeReadOnly {
		return nil
	}

	if md.deletePair() || md.state.Delete(-1) != 0 {
		return gvcode.ChangeEvent{}
	}
	return nil
}

// deletePair removes the generated closing delimiter together with its opening half.
func (md *mathDelimiters) deletePair() bool {
	opening, closing, ok := md.pair()
	if !ok {
		return false
	}

	start, end := md.state.Selection()
	if start != end || start != opening+1 {
		return false
	}

	// Delete the closing half first to keep the opening offset valid.
	md.state.SetCaret(closing, closing+1)
	md.state.Delete(1)
	md.state.SetCaret(opening, opening+1)
	md.state.Delete(1)
	md.forget()
	return true
}

// pair returns the tracked offsets, dropping them once they no longer enclose a
// math block.
func (md *mathDelimiters) pair() (int, int, bool) {
	if md.opening == nil || md.closing == nil {
		return 0, 0, false
	}

	opening, closing := md.opening.Offset(), md.closing.Offset()
	if opening < 0 || closing <= opening || closing >= md.state.Len() ||
		md.runeAt(opening) != mathDelim || md.runeAt(closing) != mathDelim {
		md.forget()
		return 0, 0, false
	}
	return opening, closing, true
}

// track marks both delimiters. The forward bias moves a marker along with the
// delimiter it points at when text is inserted in front of it.
func (md *mathDelimiters) track(opening, closing int) {
	md.forget()

	openingMarker, err := md.state.CreateMarker(opening, gvcode.BiasForward)
	if err != nil {
		return
	}
	closingMarker, err := md.state.CreateMarker(closing, gvcode.BiasForward)
	if err != nil {
		md.state.RemoveMarker(openingMarker)
		return
	}

	md.opening, md.closing = openingMarker, closingMarker
}

func (md *mathDelimiters) forget() {
	if md.opening != nil {
		md.state.RemoveMarker(md.opening)
		md.opening = nil
	}
	if md.closing != nil {
		md.state.RemoveMarker(md.closing)
		md.closing = nil
	}
}

func (md *mathDelimiters) runeAt(offset int) rune {
	if offset < 0 || offset >= md.state.Len() {
		return 0
	}

	r, _ := utf8.DecodeRuneInString(md.state.ReadTextBetween(offset, offset+1))
	return r
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
