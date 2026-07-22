package settings

import (
	"errors"
	"fmt"
	"strconv"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	gvwidget "github.com/oligo/gioview/widget"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/settings"
	"looz.ws/typstify/typst"
	"looz.ws/typstify/ui/palette"
	"looz.ws/typstify/ui/settings/form"
)

type SubSettingView interface {
	Title() string
	Layout(gtx C, th *theme.Theme) D
}

type settingItem struct {
	alignment layout.Alignment
}

type GeneralView struct {
	setting       *settings.GeneralSettings
	langPref      widget.Enum
	textSizeInput *form.FloatBinder
	typeface      gvwidget.TextField
	themeChoice   widget.Enum
	checkUpdate   widget.Bool

	externalTypstInput    gvwidget.TextField
	externalTinymistInput gvwidget.TextField
	isInitialized         bool
	lastErr               error
}

type EditorView struct {
	setting              *settings.EditorSettings
	saveIntervalInput    *form.FloatBinder
	typeface             gvwidget.TextField
	textSizeInput        *form.FloatBinder
	lineHeightScaleInput *form.FloatBinder
	previewWidthInput    *form.FloatBinder
	textWeightEnum       widget.Enum
	tabKind              widget.Enum
	tabSize              gvwidget.TextField

	isInitialized bool
	lastErr       error

	weightChoices []layout.FlexChild
	tabKindChoces []layout.FlexChild
}

type FileInterfaceView struct {
	srv              *service.ServiceFacade
	setting          *settings.FileInterfaceSettings
	mode             widget.Enum
	editorScope      widget.Enum
	libraryView      widget.Enum
	navigationLayout widget.Enum
	changeLibraryBtn widget.Clickable
	forgetLibraryBtn widget.Clickable
	isInitialized    bool
	lastErr          error
}

type TypstSettingsView struct {
	setting             *settings.TypstSettings
	cacheDirInput       gvwidget.TextField
	pkgDirInput         gvwidget.TextField
	fontPathInput       gvwidget.TextField
	outputDirInput      gvwidget.TextField
	ignoreSystemFonts   widget.Bool
	ignoreEmbeddedFonts widget.Bool
	buildDeps           widget.Bool
	useSysInputs        widget.Bool
	typstVersion        string

	isInitialized bool
	lastErr       error
}

func (item settingItem) Layout(gtx C, th *theme.Theme, title, labelDesc string, w layout.Widget) D {
	return layout.Inset{
		Bottom: unit.Dp(24),
	}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Vertical,
			Alignment: item.alignment,
		}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				lb := material.Subtitle2(th.Theme, title)
				lb.Font.Weight = font.SemiBold
				return lb.Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				if labelDesc == "" {
					return D{}
				}
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx)
			}),
			layout.Rigid(func(gtx C) D {
				if labelDesc == "" {
					return D{}
				}

				lb := material.Label(th.Theme, th.TextSize, labelDesc)
				// lb.Color = misc.WithAlpha(th.Fg, 0xb6)
				return lb.Layout(gtx)
			}),

			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx C) D {
				return layout.UniformInset(unit.Dp(3)).Layout(gtx, w)
			}),
		)
	})

}

func (g *GeneralView) Layout(gtx C, th *theme.Theme) D {
	g.typeface.SingleLine = true

	if !g.isInitialized {
		g.externalTypstInput.SetText(g.setting.ExternalTypst)
		g.externalTinymistInput.SetText(g.setting.ExternalTinymist)
		g.langPref.Value = g.setting.Language
		g.textSizeInput = form.NewFloatBinder(float32(g.setting.TextSize), []float32{10, 18})
		g.typeface.SetText(string(g.setting.TypeFace))
		g.themeChoice.Value = string(g.setting.Theme)
		g.checkUpdate = widget.Bool{Value: g.setting.CheckUpdate == "true"}

		g.isInitialized = true
	} else {
		var doUpdate bool

		if g.langPref.Update(gtx) {
			g.setting.Language = g.langPref.Value
			doUpdate = true
		}

		if val, updated := g.textSizeInput.Update(gtx); updated {
			g.setting.TextSize = val
			doUpdate = true
		}

		if g.typeface.Changed() || g.typeface.Submitted() {
			g.setting.TypeFace = g.typeface.Text()
			doUpdate = true
		}

		if g.themeChoice.Update(gtx) {
			g.setting.Theme = g.themeChoice.Value
			doUpdate = true
		}

		if g.checkUpdate.Update(gtx) {
			if g.checkUpdate.Value {
				g.setting.CheckUpdate = "true"
			} else {
				g.setting.CheckUpdate = "false"
			}
			doUpdate = true
		}

		if g.externalTypstInput.Changed() || g.externalTypstInput.Submitted() {
			g.setting.ExternalTypst = g.externalTypstInput.Text()
			doUpdate = true
		}
		if g.externalTinymistInput.Changed() || g.externalTinymistInput.Submitted() {
			g.setting.ExternalTinymist = g.externalTinymistInput.Text()
			doUpdate = true
		}

		if doUpdate {
			g.lastErr = g.setting.Save()
		}
	}

	flexChidren := make([]layout.FlexChild, 0)
	for _, locale := range i18n.Locales {
		locale := locale
		flexChidren = append(flexChidren, layout.Rigid(func(gtx C) D {
			return material.RadioButton(th.Theme, &g.langPref, locale.ID, locale.Name).Layout(gtx)
		}))
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			if g.lastErr != nil {
				return misc.LayoutErrorLabel(gtx, th, g.lastErr)

			} else {
				return layout.Dimensions{}
			}
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("Language"),
				i18n.Translate("Set the displaying language of the user interface."),
				func(gtx C) D {
					return layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx, flexChidren...)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("UI Text Size"), i18n.Translate("Set font size for the user interface. The unit is in scale-independent pixel (sp)."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Flexed(1, material.Slider(th.Theme, g.textSizeInput.GetWidget(gtx)).Layout),
						layout.Rigid(material.Body1(th.Theme, fmt.Sprintf("%.0f sp", g.textSizeInput.Value())).Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("UI Font Family"),
				i18n.Translate("Set the desired fonts for the UI. Use CSS style font family syntax which is a list of comma separated names. You can also leave it empty to let the application choose a fallback."),
				func(gtx C) D {
					g.typeface.Alignment = text.Start
					return g.typeface.Layout(gtx, th, i18n.Translate("Font Family"))
				})
		}),

		layout.Rigid(func(gtx C) D {
			choices := make([]layout.FlexChild, 0)
			for i := 0; i < len(palette.ThemeNames()); i = i + 2 {
				name := palette.ThemeNames()[i]
				row := make([]layout.FlexChild, 0)
				row = append(row, layout.Flexed(0.5, func(gtx C) D {
					return layout.Inset{Right: unit.Dp(3)}.Layout(gtx,
						material.RadioButton(th.Theme, &g.themeChoice, name, name).Layout)
				}))

				if i+1 < len(palette.ThemeNames()) {
					name2 := palette.ThemeNames()[i+1]
					row = append(row, layout.Flexed(0.5, func(gtx C) D {
						return layout.Inset{Right: unit.Dp(3)}.Layout(gtx,
							material.RadioButton(th.Theme, &g.themeChoice, name2, name2).Layout)
					}))
				}

				choices = append(choices,
					layout.Rigid(func(gtx C) D {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, row...)
					}),
				)
			}
			return settingItem{alignment: layout.Start}.Layout(gtx, th,
				i18n.Translate("Theme"), i18n.Translate("Choose your favorite color theme for the user interface."), func(gtx C) D {
					return layout.Flex{
						Axis: layout.Vertical,
					}.Layout(gtx, choices...)
				})

		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("External Typst Compiler Path"),
				i18n.Translate("Specifies a path to your own version of Typst compiler. Leave it empty to use the built-in one. Please check the compatibility before you switch. It needs to restart to take effect."),
				func(gtx C) D {
					g.externalTypstInput.Alignment = text.Start
					return g.externalTypstInput.Layout(gtx, th, "")
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("External LSP Server(Tinymist) Path"),
				i18n.Translate("Specifies a path to your own version of Tinymist. Leave it empty to use the built-in one. Please check the compatibility before you switch. It needs to restart to take effect."),
				func(gtx C) D {
					g.externalTinymistInput.Alignment = text.Start
					return g.externalTinymistInput.Layout(gtx, th, "")
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Check Update"),
				i18n.Translate("Check for updates on app startup. Please enable it to keep up to date for new features and bugfixes."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &g.checkUpdate, "Check for updates on app startup").Layout),
					)
				})
		}),
	)
}

func (g *GeneralView) Title() string { return i18n.Translate("General") }

var fontWeights = []font.Weight{font.Normal, font.Medium, font.SemiBold, font.Bold}
var tabKindOptions = []string{"Spaces", "Tab"}

func currentTabKind(s *settings.EditorSettings) string {
	if s.UseSoftTab == "true" {
		return tabKindOptions[0]
	}
	return tabKindOptions[1]
}

// func selectedSoftTab(opt string)

func (e *EditorView) Layout(gtx C, th *theme.Theme) D {
	e.typeface.SingleLine = true
	e.tabSize.SingleLine = true

	if !e.isInitialized {
		e.textSizeInput = form.NewFloatBinder(float32(e.setting.TextSize), []float32{6, 36})
		e.lineHeightScaleInput = form.NewFloatBinder(e.setting.LineHeightScale*10, []float32{10, 30})
		e.typeface.SetText(string(e.setting.TypeFace))
		e.textWeightEnum.Value = fmt.Sprint(int(e.setting.Weight))
		e.tabKind.Value = currentTabKind(e.setting)
		e.tabSize.SetText(fmt.Sprint(e.setting.TabSize))
		e.saveIntervalInput = form.NewFloatBinder(float32(e.setting.AutoSaveInterval), []float32{1, 10})
		e.previewWidthInput = form.NewFloatBinder(float32(e.setting.PreviewWidth), []float32{20, 60})
		e.isInitialized = true
	} else {
		var doUpdate bool

		if val, updated := e.saveIntervalInput.Update(gtx); updated {
			e.setting.AutoSaveInterval = int(val)
			doUpdate = true
		}

		if e.typeface.Changed() || e.typeface.Submitted() {
			e.setting.TypeFace = e.typeface.Text()
			doUpdate = true
		}

		if val, updated := e.textSizeInput.Update(gtx); updated {
			e.setting.TextSize = val
			doUpdate = true
		}

		if val, updated := e.lineHeightScaleInput.Update(gtx); updated {
			e.setting.LineHeightScale = val / 10.0
			doUpdate = true
		}

		if val, updated := e.previewWidthInput.Update(gtx); updated {
			e.setting.PreviewWidth = int(val)
			doUpdate = true
		}

		if e.textWeightEnum.Update(gtx) {
			val, err := strconv.Atoi(e.textWeightEnum.Value)
			if err != nil {
				panic(err)
			}
			e.setting.Weight = val
			doUpdate = true
		}

		if e.tabKind.Update(gtx) {
			switch e.tabKind.Value {
			case tabKindOptions[0]:
				e.setting.UseSoftTab = "true"
			default:
				e.setting.UseSoftTab = "false"
			}
			doUpdate = true
		}

		if e.tabSize.Changed() || e.tabSize.Submitted() {
			tabSize, err := strconv.Atoi(e.tabSize.Text())
			if err != nil {
				e.lastErr = errors.Unwrap(err)
				doUpdate = false
			} else {
				e.setting.TabSize = tabSize
				doUpdate = true
			}
		}

		if doUpdate {
			e.lastErr = e.setting.Save()
		}
	}

	if len(e.weightChoices) == 0 {
		for _, weight := range fontWeights {
			weight := weight
			e.weightChoices = append(e.weightChoices, layout.Rigid(func(gtx C) D {
				return material.RadioButton(th.Theme, &e.textWeightEnum, fmt.Sprint(int(weight)), weight.String()).Layout(gtx)
			}))
		}
	}

	if len(e.tabKindChoces) == 0 {
		for _, kind := range tabKindOptions {
			e.tabKindChoces = append(e.tabKindChoces, layout.Rigid(func(gtx C) D {
				return material.RadioButton(th.Theme, &e.tabKind, kind, kind).Layout(gtx)
			}))
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			if e.lastErr != nil {
				return misc.LayoutErrorLabel(gtx, th, e.lastErr)
			} else {
				return layout.Dimensions{}
			}
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("Auto Save Interval"),
				i18n.Translate("Set the auto save delay (in seconds), cannot be completely disabled."),

				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Flexed(1, material.Slider(th.Theme, e.saveIntervalInput.GetWidget(gtx)).Layout),
						layout.Rigid(material.Body1(th.Theme, fmt.Sprintf("%.0fs", e.saveIntervalInput.Value())).Layout),
					)
				})

		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("Preview width"),
				i18n.Translate("Set the width used whenever the inline preview opens. Quarter is 25%, third is 33%, and half is 50%. You can still drag the divider while editing."),
				func(gtx C) D {
					previewWidth := int(e.previewWidthInput.Value())
					return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
						layout.Flexed(1, material.Slider(th.Theme, e.previewWidthInput.GetWidget(gtx)).Layout),
						layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
						layout.Rigid(material.Body1(th.Theme, fmt.Sprintf("%d%%", previewWidth)).Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("Font Family"),
				i18n.Translate("Set the desired fonts for the editor. Use CSS style font family syntax which is a list of comma separated names. You can also leave it empty to let the application choose a fallback."),
				func(gtx C) D {
					e.typeface.Alignment = text.Start
					return e.typeface.Layout(gtx, th, i18n.Translate("Font Family"))
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Text Size"),
				i18n.Translate("Set font size for the editor. The unit is in scale-independent pixel (sp)."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Flexed(1, material.Slider(th.Theme, e.textSizeInput.GetWidget(gtx)).Layout),
						layout.Rigid(material.Body1(th.Theme, fmt.Sprintf("%.0f sp", e.textSizeInput.Value())).Layout),
					)
				})

		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("Font Weight"),
				i18n.Translate("Set the expected weight (or boldness) of the text. This is for variable fonts only."),
				func(gtx C) D {
					return layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx, e.weightChoices...)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Line Height Scale"),
				i18n.Translate("Set the line height scale of the lines in editor. Line height scale is multiplied by line height to determine the final gap between lines."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Flexed(1, material.Slider(th.Theme, e.lineHeightScaleInput.GetWidget(gtx)).Layout),
						layout.Rigid(material.Body1(th.Theme, fmt.Sprintf("%.1f", e.lineHeightScaleInput.Value()/10)).Layout),
					)
				})

		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("Indentation"),
				i18n.Translate("Set the expected characters to use when pressing the Tab key. Please be noted that this works only for empty file. Indentation for non-empty files are auto detected."),
				func(gtx C) D {
					return layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx, e.tabKindChoces...)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Tab Width"),
				i18n.Translate("Set how many number of spaces the Tab is equal to. Please be noted that this works only for empty file. Indentation for non-empty files are auto detected."),
				func(gtx C) D {
					e.tabSize.Alignment = text.Start
					return e.tabSize.Layout(gtx, th, i18n.Translate("Tab Width"))
				})
		}),
	)
}

func (e *EditorView) Title() string { return i18n.Translate("Editor") }

func (f *FileInterfaceView) Title() string { return i18n.Translate("File Interface") }

func (f *FileInterfaceView) Layout(gtx C, th *theme.Theme) D {
	if f.changeLibraryBtn.Clicked(gtx) {
		go func() {
			path, err := f.srv.FileChooser().ChooseFolder()
			if err != nil || path == "" {
				return
			}
			f.setting.LibraryRoot = path
			f.lastErr = f.setting.Save()
		}()
	}
	if f.forgetLibraryBtn.Clicked(gtx) {
		f.setting.LibraryRoot = ""
		f.lastErr = f.setting.Save()
	}

	if !f.isInitialized {
		f.mode.Value = string(f.setting.Mode)
		f.editorScope.Value = string(f.setting.EditorScope)
		f.libraryView.Value = string(f.setting.LibraryView)
		f.navigationLayout.Value = string(f.setting.NavigationLayout)
		f.isInitialized = true
	} else {
		var doUpdate bool
		if f.mode.Update(gtx) {
			f.setting.Mode = settings.FileInterfaceMode(f.mode.Value)
			doUpdate = true
		}
		if f.editorScope.Update(gtx) {
			f.setting.EditorScope = settings.FileInterfaceEditorScope(f.editorScope.Value)
			doUpdate = true
		}
		if f.libraryView.Update(gtx) {
			f.setting.LibraryView = settings.FileInterfaceLibraryView(f.libraryView.Value)
			doUpdate = true
		}
		if f.navigationLayout.Update(gtx) {
			f.setting.NavigationLayout = settings.FileInterfaceNavigationLayout(f.navigationLayout.Value)
			doUpdate = true
		}
		if doUpdate {
			f.lastErr = f.setting.Save()
		}
	}

	children := []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			if f.lastErr != nil {
				return misc.LayoutErrorLabel(gtx, th, f.lastErr)
			}
			return D{}
		}),
		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th,
				i18n.Translate("Interface"),
				i18n.Translate("Classic keeps Typstify's existing file workflow. Student adds a Library home and focused project navigation."),
				func(gtx C) D {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(material.RadioButton(th.Theme, &f.mode, string(settings.FileInterfaceModeClassic), i18n.Translate("Classic")).Layout),
						layout.Rigid(material.RadioButton(th.Theme, &f.mode, string(settings.FileInterfaceModeStudent), i18n.Translate("Student")).Layout),
					)
				})
		}),
	}

	if f.mode.Value != string(settings.FileInterfaceModeStudent) {
		children = append(children, layout.Rigid(func(gtx C) D {
			label := material.Caption(th.Theme, i18n.Translate("Your Student preferences are saved and will return when you choose Student."))
			label.Color = misc.WithAlpha(th.Fg, 0x90)
			return label.Layout(gtx)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	}

	children = append(children, layout.Rigid(func(gtx C) D {
		return widget.Border{Color: misc.WithAlpha(th.Fg, 0x30), Width: unit.Dp(1), CornerRadius: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
			return layout.Inset{Top: unit.Dp(18), Bottom: unit.Dp(2), Left: unit.Dp(18), Right: unit.Dp(18)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						label := material.H6(th.Theme, i18n.Translate("Student preferences"))
						label.Font.Weight = font.SemiBold
						return label.Layout(gtx)
					}),
					layout.Rigid(func(gtx C) D {
						label := material.Caption(th.Theme, i18n.Translate("These choices apply immediately and do not change the Classic interface."))
						label.Color = misc.WithAlpha(th.Fg, 0xa0)
						return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(18)}.Layout(gtx, label.Layout)
					}),
					layout.Rigid(func(gtx C) D {
						return settingItem{}.Layout(gtx, th,
							i18n.Translate("Library folder"),
							i18n.Translate("The folder shown when Student mode starts. Forgetting it does not delete any files."),
							func(gtx C) D {
								path := f.setting.LibraryRoot
								if path == "" {
									path = i18n.Translate("No Library folder selected")
								}
								return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
									layout.Flexed(1, func(gtx C) D {
										label := material.Body2(th.Theme, path)
										label.MaxLines = 2
										return label.Layout(gtx)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
									layout.Rigid(func(gtx C) D {
										text := i18n.Translate("Choose folder")
										if f.setting.LibraryRoot != "" {
											text = i18n.Translate("Change")
										}
										return material.Button(th.Theme, &f.changeLibraryBtn, text).Layout(gtx)
									}),
									layout.Rigid(func(gtx C) D {
										if f.setting.LibraryRoot == "" {
											return D{}
										}
										return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
											button := material.Button(th.Theme, &f.forgetLibraryBtn, i18n.Translate("Forget"))
											button.Background = th.Bg
											button.Color = th.Fg
											return button.Layout(gtx)
										})
									}),
								)
							})
					}),
					layout.Rigid(func(gtx C) D {
						return settingItem{}.Layout(gtx, th,
							i18n.Translate("Editor files"),
							i18n.Translate("Show only the current project or folder, or keep the complete workspace tree."),
							func(gtx C) D {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(material.RadioButton(th.Theme, &f.editorScope, string(settings.FileInterfaceEditorScopeContextual), i18n.Translate("Current project or folder")).Layout),
									layout.Rigid(material.RadioButton(th.Theme, &f.editorScope, string(settings.FileInterfaceEditorScopeWholeWorkspace), i18n.Translate("Whole workspace")).Layout),
								)
							})
					}),
					layout.Rigid(func(gtx C) D {
						return settingItem{}.Layout(gtx, th,
							i18n.Translate("Library view"),
							i18n.Translate("Choose the default presentation for folders, notebooks, and files."),
							func(gtx C) D {
								return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
									layout.Rigid(material.RadioButton(th.Theme, &f.libraryView, string(settings.FileInterfaceLibraryViewGrid), i18n.Translate("Grid")).Layout),
									layout.Rigid(material.RadioButton(th.Theme, &f.libraryView, string(settings.FileInterfaceLibraryViewList), i18n.Translate("List")).Layout),
								)
							})
					}),
					layout.Rigid(func(gtx C) D {
						return settingItem{}.Layout(gtx, th,
							i18n.Translate("Navigation"),
							i18n.Translate("Use the Library rail, or keep Typstify's familiar sidebar and bottom controls."),
							func(gtx C) D {
								return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
									layout.Rigid(material.RadioButton(th.Theme, &f.navigationLayout, string(settings.FileInterfaceNavigationLayoutLibrary), i18n.Translate("Library rail")).Layout),
									layout.Rigid(material.RadioButton(th.Theme, &f.navigationLayout, string(settings.FileInterfaceNavigationLayoutClassic), i18n.Translate("Classic controls")).Layout),
								)
							})
					}),
				)
			})
		})
	}))

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (t *TypstSettingsView) Title() string { return i18n.Translate("Typst") }

func (t *TypstSettingsView) Layout(gtx C, th *theme.Theme) D {
	if !t.isInitialized {
		t.cacheDirInput.SetText(t.setting.PackageCacheDir)
		t.pkgDirInput.SetText(t.setting.PackageDir)
		t.fontPathInput.SetText(t.setting.ExtraFontPath)
		t.outputDirInput.SetText(t.setting.OutputDir)
		t.typstVersion = typst.CurrentVersion()
		t.ignoreSystemFonts = widget.Bool{Value: t.setting.IgnoreSystemFonts != 0}
		t.ignoreEmbeddedFonts = widget.Bool{Value: t.setting.IgnoreEmbeddedFonts != 0}
		t.useSysInputs = widget.Bool{Value: t.setting.UseSysInputs != 0}
		t.buildDeps = widget.Bool{Value: t.setting.BuildDeps != 0}
		t.isInitialized = true
	} else {
		var doUpdate bool
		if t.cacheDirInput.Changed() || t.cacheDirInput.Submitted() {
			t.setting.PackageCacheDir = t.cacheDirInput.Text()
			doUpdate = true
		}

		if t.pkgDirInput.Changed() || t.pkgDirInput.Submitted() {
			t.setting.PackageDir = t.pkgDirInput.Text()
			doUpdate = true
		}

		if t.fontPathInput.Changed() || t.fontPathInput.Submitted() {
			t.setting.ExtraFontPath = t.fontPathInput.Text()
			doUpdate = true
		}

		if t.outputDirInput.Changed() || t.outputDirInput.Submitted() {
			t.setting.OutputDir = t.outputDirInput.Text()
			doUpdate = true
		}

		if t.ignoreSystemFonts.Update(gtx) {
			if t.ignoreSystemFonts.Value {
				t.setting.IgnoreSystemFonts = 1
			} else {
				t.setting.IgnoreSystemFonts = 0
			}

			doUpdate = true
		}

		if t.ignoreEmbeddedFonts.Update(gtx) {
			if t.ignoreEmbeddedFonts.Value {
				t.setting.IgnoreEmbeddedFonts = 1
			} else {
				t.setting.IgnoreEmbeddedFonts = 0
			}

			doUpdate = true
		}

		if t.buildDeps.Update(gtx) {
			if t.buildDeps.Value {
				t.setting.BuildDeps = 1
			} else {
				t.setting.BuildDeps = 0
			}
			doUpdate = true
		}

		if t.useSysInputs.Update(gtx) {
			if t.useSysInputs.Value {
				t.setting.UseSysInputs = 1
			} else {
				t.setting.UseSysInputs = 0
			}
		}

		if doUpdate {
			t.lastErr = t.setting.Save()
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			if t.lastErr != nil {
				return misc.LayoutErrorLabel(gtx, th, t.lastErr)
			} else {
				return layout.Dimensions{}
			}
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Version"),
				"",
				func(gtx C) D {
					return material.Label(th.Theme, th.TextSize, fmt.Sprintf("Typst:    %s", t.typstVersion)).Layout(gtx)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Package Dir"),
				i18n.Translate("Specifies where to store your local Typst packages/templates. Leave it empty to use the default dir."),
				func(gtx C) D {
					t.pkgDirInput.Alignment = text.Start
					return t.pkgDirInput.Layout(gtx, th, "")
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Package Cache Dir"),
				i18n.Translate("Specifies where to store your the cached Typst packages/templates retrieved from the network. Leave it empty to use the default dir."),
				func(gtx C) D {
					t.cacheDirInput.Alignment = text.Start
					return t.cacheDirInput.Layout(gtx, th, "")
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Extra Font Path"),
				i18n.Translate("The directory where to search for fonts when exporting, previewing and auto-completing. Be aware that the current project root directory is always searched. Need to restart or reload to take effect."),
				func(gtx C) D {
					t.fontPathInput.Alignment = text.Start
					return t.fontPathInput.Layout(gtx, th, "")
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Export Dir"),
				i18n.Translate("The directory to save exported files to, including the deps.json. If not set, the exported files will be saved to a 'output' folder beside the source file."),
				func(gtx C) D {
					t.outputDirInput.Alignment = text.Start
					return t.outputDirInput.Layout(gtx, th, "")
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Load External Inputs"),
				i18n.Translate(`Load external inputs from a file named sys-inputs.json as sys.inputs. If there is no such one, it is created at the root dir. 
A sys-inputs.json file contains user defined key-value pairs which can be accessed via Typst's sys.inputs. The values should always be string encoded data.
Need to restart or reload to take effect for code linter, auto-completion, and preview when changed.`),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &t.useSysInputs, "Load sys-inputs.json").Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Ignore System Fonts"),
				i18n.Translate("Ignore system fonts or not. For code linter, auto-completion and previewing, it needs to restart or reload to take effect."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &t.ignoreSystemFonts, "Ignore system fonts").Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Ignore Compiler Embedded Fonts"),
				i18n.Translate("Ignore embedded Fonts or not. This only works when exporting files."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &t.ignoreEmbeddedFonts, "Ignore embedded fonts").Layout),
					)
				})
		}),

		layout.Rigid(func(gtx C) D {
			return settingItem{}.Layout(gtx, th, i18n.Translate("Generate Dependencies file"),
				i18n.Translate("Write dependencies of the file compiled to a file named deps.json in your project directory."),
				func(gtx C) D {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(material.Switch(th.Theme, &t.buildDeps, "Generate deps file").Layout),
					)
				})
		}),
	)
}
