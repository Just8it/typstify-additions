package editor

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/oligo/gvcode"
)

// EditorListeners for TextEditor. An listener observe editor state changes and react to
// update UI, other state, etc.
//
// This should be set per TextEditor instance by the view layer.
type EditorListeners struct {
	// Listener for text selection changes
	OnSelectChange func(gvcode.Position)

	// Listener for editor content changes.
	OnTextChange func()

	// Listener for link clicking in the editor and hover tips.
	OnOpenLink func(link string, external bool)
}

// TextEditorOption defines options to configure various aspects of the editor.
type TextEditorOption func(editor *TextEditor)

// WithListeners set event listeners for the editor. Should be set per-instance.
func WithListeners(listeners EditorListeners) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.listeners = listeners
	}
}

func WithEnterHook(hook gvcode.EnterHook) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.AddEnterHook(hook))
	}
}

func WithTextInputHook(hook gvcode.TextInputHook) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.AddTextInputHook(hook))
	}
}

func WithPasteHook(hook gvcode.BeforePasteHook) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.AddBeforePasteHook(hook))
	}
}

// WithBracketPairs configures bracket pairs (e.g. "()", "{}", "[]") for auto-completion.
// When the left half of a pair is entered, the right half is automatically inserted.
func WithBracketPairs(bracketPairs map[rune]rune) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.WithBracketPairs(bracketPairs))
	}
}

// WithQuotePairs configures quote pairs (e.g. '"', ''', '`') for auto-completion.
// When the left half of a pair is entered, the right half is automatically inserted.
func WithQuotePairs(quotePairs map[rune]rune) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.WithQuotePairs(quotePairs))
	}
}

// WithWordSeparators configures the set of runes treated as word boundaries for
// word-based navigation (Ctrl+Left/Right) and deletion (Ctrl+Backspace/Delete).
func WithWordSeparators(separators string) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.WithWordSeperators(separators))
	}
}

// WithSoftTab controls whether the editor inserts spaces (true) or a tab character (false)
// when the Tab key is pressed.
func WithSoftTab(enabled bool) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.WithSoftTab(enabled))
	}
}

// WithTabWidth sets the number of spaces a tab character represents. For soft tabs,
// this is the number of spaces inserted. For hard tabs, this controls the display width.
func WithTabWidth(width int) TextEditorOption {
	return func(editor *TextEditor) {
		if editor.state == nil {
			panic("editor is not initialized")
		}

		editor.state.WithOptions(gvcode.WithTabWidth(width))
	}
}

// LanguageExtension defines an extention for a programming language for the editor.
type LanguageExtension struct {
	// Lang is the canonical programming language name.
	Lang string
	// Exts provides file extensions which should be recognized as the language source file.
	Exts []string
	// Options to configure the language editing behaviour. Listener option should not
	// be registered as a language option here.
	Opts []TextEditorOption
}

type languageExtRegistry struct {
	mu sync.RWMutex

	// map file extension (e.g. ".typ") to options
	extensions map[string]LanguageExtension
}

// A package-level singleton to manage language extensions.
var langExtRegistry = &languageExtRegistry{
	extensions: make(map[string]LanguageExtension),
}

// optionsFor returns the registered options for the file identified by path,
// matched by file extension.
func (r *languageExtRegistry) optionsFor(path string) []TextEditorOption {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext := strings.ToLower(filepath.Ext(path))
	for _, langExt := range r.extensions {
		for _, e := range langExt.Exts {
			if strings.EqualFold(e, ext) {
				return langExt.Opts
			}
		}
	}
	return nil
}

// Register editor hooks and other options for the editor to customize the
// editing behaviour. Use this function to register language extensions.
//
// Example:
//
//	editor.RegisterLanguageExtension(editor.LanguageExtension{
//		Lang: "Typst",
//		Exts: []string{".typ"},
//		Opts: []editor.TextEditorOption{
//			editor.WithEnterHook(typstAutoCompleteEnter),
//			editor.WithTextInputHook(typstMathInputHook),
//		},
//	})
func RegisterLanguageExtension(langExt LanguageExtension) {
	langExtRegistry.mu.Lock()
	defer langExtRegistry.mu.Unlock()

	if langExt.Lang == "" || len(langExt.Exts) == 0 {
		panic("unknown language extention")
	}

	langExtRegistry.extensions[langExt.Lang] = langExt

}
