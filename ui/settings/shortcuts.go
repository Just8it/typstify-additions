package settings

import (
	"image/color"
	"strings"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	gvwidget "github.com/oligo/gioview/widget"

	"looz.ws/typstify/i18n"
	config "looz.ws/typstify/service/settings"
)

var shortcutWarning = color.NRGBA{R: 0xe0, G: 0x9f, B: 0x3e, A: 0xff}

type ShortcutsView struct {
	setting   *config.EditorSettings
	search    gvwidget.TextField
	resetAll  widget.Clickable
	rows      map[config.ShortcutID]*shortcutRow
	recording config.ShortcutID
	lastErr   error
}

type shortcutRow struct {
	enabled widget.Bool
	capture widget.Clickable
	reset   widget.Clickable
	warning string
}

func (view *ShortcutsView) Title() string { return i18n.Translate("Shortcuts") }

func (view *ShortcutsView) Layout(gtx C, th *theme.Theme) D {
	view.initialize()
	view.search.SingleLine = true

	if view.resetAll.Clicked(gtx) {
		view.lastErr = view.setting.ResetShortcuts()
		view.recording = ""
		view.syncRows()
	}

	children := []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Flexed(1, func(gtx C) D {
						return view.search.Layout(gtx, th, i18n.Translate("Search actions or shortcuts"))
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
					layout.Rigid(func(gtx C) D {
						return material.Button(th.Theme, &view.resetAll, i18n.Translate("Restore all defaults")).Layout(gtx)
					}),
				)
			})
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Bottom: unit.Dp(20)}.Layout(gtx, func(gtx C) D {
				label := material.Body2(th.Theme, i18n.Translate("Click a shortcut field, then press the new key combination. Escape cancels recording."))
				label.Color = misc.WithAlpha(th.Fg, 0xa0)
				return label.Layout(gtx)
			})
		}),
	}

	if view.lastErr != nil {
		children = append(children, layout.Rigid(func(gtx C) D {
			return misc.LayoutErrorLabel(gtx, th, view.lastErr)
		}))
	}

	query := strings.ToLower(strings.TrimSpace(view.search.Text()))
	category := ""
	for _, definition := range config.ShortcutDefinitions() {
		if !view.matches(definition, query) {
			continue
		}
		if definition.Category != category {
			category = definition.Category
			name := category
			children = append(children, layout.Rigid(func(gtx C) D {
				return layout.Inset{Top: unit.Dp(10), Bottom: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
					label := material.Label(th.Theme, th.TextSize*0.85, i18n.Translate(name))
					label.Font.Weight = 600
					label.Color = misc.WithAlpha(th.Fg, 0xc0)
					return label.Layout(gtx)
				})
			}))
		}
		definition := definition
		children = append(children, layout.Rigid(func(gtx C) D {
			return view.layoutRow(gtx, th, definition)
		}))
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (view *ShortcutsView) initialize() {
	if view.rows != nil {
		return
	}
	view.rows = make(map[config.ShortcutID]*shortcutRow)
	for _, definition := range config.ShortcutDefinitions() {
		view.rows[definition.ID] = &shortcutRow{enabled: widget.Bool{Value: !config.ShortcutDisabled(view.setting, definition.ID)}}
	}
}

func (view *ShortcutsView) syncRows() {
	for id, row := range view.rows {
		row.enabled.Value = !config.ShortcutDisabled(view.setting, id)
		row.warning = ""
	}
}

func (view *ShortcutsView) matches(definition config.ShortcutDefinition, query string) bool {
	if query == "" {
		return true
	}
	binding, enabled := config.EffectiveShortcut(view.setting, definition.ID)
	shortcut := i18n.Translate("Disabled")
	if enabled {
		shortcut = binding.Display()
	}
	text := definition.Category + " " + definition.Name + " " + definition.Description + " " + shortcut
	return strings.Contains(strings.ToLower(text), query)
}

func (view *ShortcutsView) layoutRow(gtx C, th *theme.Theme, definition config.ShortcutDefinition) D {
	row := view.rows[definition.ID]
	view.updateRow(gtx, definition, row)

	borderColor := misc.WithAlpha(th.Fg, 0x30)
	if view.recording == definition.ID {
		borderColor = th.ContrastBg
	} else if row.warning != "" {
		borderColor = shortcutWarning
	}

	return layout.Inset{Bottom: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
		return widget.Border{Color: borderColor, Width: unit.Dp(1), CornerRadius: unit.Dp(7)}.Layout(gtx, func(gtx C) D {
			return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(14), Right: unit.Dp(12)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
							layout.Flexed(1, func(gtx C) D {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(func(gtx C) D {
										label := material.Label(th.Theme, th.TextSize, i18n.Translate(definition.Name))
										label.Font.Weight = 500
										return label.Layout(gtx)
									}),
									layout.Rigid(func(gtx C) D {
										label := material.Label(th.Theme, th.TextSize*0.78, i18n.Translate(definition.Description))
										label.Color = misc.WithAlpha(th.Fg, 0x90)
										return label.Layout(gtx)
									}),
								)
							}),
							layout.Rigid(func(gtx C) D {
								return material.Switch(th.Theme, &row.enabled, "Enable "+definition.Name).Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
							layout.Rigid(func(gtx C) D {
								return view.layoutCapture(gtx, th, definition, row)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx C) D {
								return view.layoutReset(gtx, th, definition, row)
							}),
						)
					}),
					layout.Rigid(func(gtx C) D {
						if row.warning == "" {
							return D{}
						}
						return layout.Inset{Top: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
							label := material.Label(th.Theme, th.TextSize*0.8, row.warning)
							label.Color = shortcutWarning
							return label.Layout(gtx)
						})
					}),
				)
			})
		})
	})
}

func (view *ShortcutsView) updateRow(gtx C, definition config.ShortcutDefinition, row *shortcutRow) {
	if row.enabled.Update(gtx) {
		view.lastErr = view.setting.SetShortcutEnabled(definition.ID, row.enabled.Value)
		row.warning = ""
		if !row.enabled.Value && view.recording == definition.ID {
			view.recording = ""
		}
	}
	if row.reset.Clicked(gtx) && config.ShortcutHasOverride(view.setting, definition.ID) {
		view.lastErr = view.setting.ResetShortcut(definition.ID)
		row.enabled.Value = true
		row.warning = ""
		if view.recording == definition.ID {
			view.recording = ""
		}
	}
	if row.capture.Clicked(gtx) && row.enabled.Value {
		view.recording = definition.ID
		row.warning = ""
		gtx.Execute(key.FocusCmd{Tag: row})
	}

	if view.recording != definition.ID {
		return
	}
	event.Op(gtx.Ops, row)
	if !gtx.Focused(row) {
		gtx.Execute(key.FocusCmd{Tag: row})
	}
	filters := shortcutRecorderFilters(row)
	for {
		event, ok := gtx.Event(filters...)
		if !ok {
			break
		}
		switch event := event.(type) {
		case key.FocusEvent:
			if !event.Focus {
				view.recording = ""
			}
		case key.Event:
			if event.State != key.Press {
				continue
			}
			if event.Name == key.NameEscape && event.Modifiers == 0 {
				view.recording = ""
				row.warning = ""
				continue
			}
			binding, valid := config.ShortcutBindingFromEvent(event)
			if !valid {
				continue
			}
			if conflict, exists := config.ShortcutConflict(view.setting, definition.ID, binding); exists {
				row.warning = i18n.Translate("%s is already used by %s.", binding.Display(), conflict.Name)
				continue
			}
			view.lastErr = view.setting.SetShortcut(definition.ID, binding)
			if view.lastErr == nil {
				view.recording = ""
				row.warning = ""
				gtx.Execute(key.FocusCmd{})
			}
		}
	}
}

func shortcutRecorderFilters(target event.Tag) []event.Filter {
	const modifiers = key.ModCtrl | key.ModCommand | key.ModShift | key.ModAlt | key.ModSuper
	filters := []event.Filter{key.FocusFilter{Target: target}}
	for name := '!'; name <= '~'; name++ {
		filters = append(filters, key.Filter{Focus: target, Name: key.Name(string(name)), Optional: modifiers})
	}
	for _, name := range []key.Name{
		key.NameEnter, key.NameReturn, key.NameEscape, key.NameTab, key.NameSpace,
		key.NameLeftArrow, key.NameRightArrow, key.NameUpArrow, key.NameDownArrow,
		key.NameHome, key.NameEnd, key.NameDeleteBackward, key.NameDeleteForward,
		key.NamePageUp, key.NamePageDown,
		key.NameF1, key.NameF2, key.NameF3, key.NameF4, key.NameF5, key.NameF6,
		key.NameF7, key.NameF8, key.NameF9, key.NameF10, key.NameF11, key.NameF12,
	} {
		filters = append(filters, key.Filter{Focus: target, Name: name, Optional: modifiers})
	}
	return append(filters, key.Filter{Focus: target})
}

func (view *ShortcutsView) layoutCapture(gtx C, th *theme.Theme, definition config.ShortcutDefinition, row *shortcutRow) D {
	gtx.Constraints.Min.X = gtx.Dp(unit.Dp(172))
	gtx.Constraints.Max.X = gtx.Constraints.Min.X
	text := i18n.Translate("Disabled")
	textColor := misc.WithAlpha(th.Fg, 0x70)
	if row.enabled.Value {
		if view.recording == definition.ID {
			text = i18n.Translate("Press shortcut…")
			textColor = th.ContrastBg
		} else if custom, exists := config.ShortcutCustomBinding(view.setting, definition.ID); exists {
			text = custom.Display()
			textColor = th.Fg
		} else {
			text = definition.Default.Display()
			textColor = misc.WithAlpha(th.Fg, 0x78)
		}
	}

	return row.capture.Layout(gtx, func(gtx C) D {
		borderColor := misc.WithAlpha(th.Fg, 0x45)
		if view.recording == definition.ID {
			borderColor = th.ContrastBg
		}
		return widget.Border{Color: borderColor, Width: unit.Dp(1), CornerRadius: unit.Dp(5)}.Layout(gtx, func(gtx C) D {
			return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
				label := material.Label(th.Theme, th.TextSize*0.9, text)
				label.Color = textColor
				return layout.Center.Layout(gtx, label.Layout)
			})
		})
	})
}

func (view *ShortcutsView) layoutReset(gtx C, th *theme.Theme, definition config.ShortcutDefinition, row *shortcutRow) D {
	customized := config.ShortcutHasOverride(view.setting, definition.ID)
	return row.reset.Layout(gtx, func(gtx C) D {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
			label := material.Label(th.Theme, th.TextSize*0.85, i18n.Translate("Reset"))
			label.Color = misc.WithAlpha(th.Fg, 0x55)
			if customized {
				label.Color = th.ContrastBg
			}
			return label.Layout(gtx)
		})
	})
}
