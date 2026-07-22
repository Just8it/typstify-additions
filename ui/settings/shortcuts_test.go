package settings

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	config "looz.ws/typstify/service/settings"
)

func TestShortcutDefinitionsHaveLocalizedLabels(t *testing.T) {
	for _, definition := range config.ShortcutDefinitions() {
		if shortcutCategoryLabel(definition.Category) == "" ||
			shortcutNameLabel(definition.ID) == string(definition.ID) ||
			shortcutDescriptionLabel(definition.ID) == "" {
			t.Fatalf("shortcut %q is missing localized copy", definition.ID)
		}
	}
}

func TestShortcutRecorderUsesCaptureButtonFocus(t *testing.T) {
	var router input.Router
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Source:      router.Source(),
		Constraints: layout.Exact(image.Pt(200, 40)),
	}
	view := ShortcutsView{setting: &config.EditorSettings{}}
	view.initialize()
	definition := config.ShortcutDefinitions()[0]
	row := view.rows[definition.ID]

	layoutCapture := func() {
		row.capture.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		})
	}
	layoutCapture()
	router.Frame(gtx.Ops)

	row.capture.Click()
	gtx.Reset()
	view.updateRow(gtx, definition, row)
	layoutCapture()
	router.Frame(gtx.Ops)

	gtx.Reset()
	view.updateRow(gtx, definition, row)
	layoutCapture()
	router.Frame(gtx.Ops)
	router.Queue(key.Event{Name: "Y", Modifiers: key.ModCtrl | key.ModShift, State: key.Press})

	gtx.Reset()
	view.updateRow(gtx, definition, row)
	binding, ok := config.ShortcutCustomBinding(view.setting, definition.ID)
	if !ok || binding.Name != "Y" || binding.Modifiers != key.ModCtrl|key.ModShift {
		t.Fatalf("captured shortcut = %+v, %t", binding, ok)
	}
}
