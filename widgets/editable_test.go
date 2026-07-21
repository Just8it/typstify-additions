package widgets

import (
	"image"
	"testing"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget/material"
)

func TestEditableHandlesChangeBeforeSubmit(t *testing.T) {
	var router input.Router
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(200, 40)),
		Source:      router.Source(),
	}
	editable := EditableLabel("new folder")
	editable.SetEditing(true)
	var changed string
	editable.OnChanged = func(text string) error {
		changed = text
		return nil
	}

	editable.Layout(gtx, material.NewTheme())
	router.Frame(gtx.Ops)
	router.Queue(key.EditEvent{
		Range: key.Range{Start: 0, End: len("new folder")},
		Text:  "Coursework\n",
	})
	editable.Update(gtx)

	if changed != "Coursework" || editable.IsEditing() {
		t.Fatalf("submit produced %q, editing=%v", changed, editable.IsEditing())
	}
}
