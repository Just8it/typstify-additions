package ui

import (
	"fmt"
	"image"
	"path/filepath"

	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	settingsmodel "looz.ws/typstify/service/settings"
	"looz.ws/typstify/ui/assistant"
	"looz.ws/typstify/ui/navpanel"
	"looz.ws/typstify/ui/preview"
	"looz.ws/typstify/ui/settings"
	"looz.ws/typstify/ui/statusbar"
	"looz.ws/typstify/widgets"
	"looz.ws/typstify/widgets/console"
	"looz.ws/typstify/widgets/icons"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

var libraryPromptIcon = icons.NewSvgIcon(icons.Home)

// previewable is implemented by views that support an inline preview panel.
type previewable interface {
	IsVisible() bool
	SetPreviewer(previewer *preview.Previewer)
	LayoutPreview(gtx C, th *theme.Theme) D
}

type HomeView struct {
	view.ViewManager
	srv          *service.ServiceFacade
	sidebar      *navpanel.NavDrawer
	tabbar       *navpanel.Tabbar
	statusBar    *statusbar.StatusBar
	consolePanel *console.Console
	menuPanel    *navpanel.MenuPanel
	library      *navpanel.Library

	studentMode      bool
	libraryActive    bool
	libraryRoot      string
	libraryRequested bool

	// horizontal resizer
	resizer         widgets.Resize
	lastResizeWidth int
	lastResizeRatio float32
	bar             *widgets.ResizeBar

	// console and main view resizer
	yresizer   *widgets.Resize
	lastYRatio float32
	hbar       *widgets.ResizeBar

	// preview resizer (view | preview split)
	previewResizer *widgets.Resize
	previewBar     *widgets.ResizeBar
	previewer      *preview.Previewer
	previewVisible bool

	welcome          WelcomeView
	accountClick     widget.Clickable
	chooseLibraryBtn widget.Clickable
}

func (hv *HomeView) ID() string {
	return "Home"
}

func (hv *HomeView) RequestSwitch(intent view.Intent) error {
	if !intent.ShowAsModal {
		hv.libraryActive = false
	}
	return hv.ViewManager.RequestSwitch(intent)
}

func (hv *HomeView) SwitchTab(idx int) {
	hv.libraryActive = false
	hv.ViewManager.SwitchTab(idx)
}

func (hv *HomeView) showLibrary() {
	if !hv.studentMode {
		return
	}
	root := hv.srv.Settings().FileInterface().LibraryRoot
	hv.libraryRoot = root
	hv.libraryActive = true
	hv.menuPanel.IsDrawerHidden = true
	if root != "" && filepath.Clean(hv.srv.Workspace().Current().Path) != filepath.Clean(root) {
		hv.srv.EventBus().Emit(bus.TopicProjectSwitched, root)
		return
	}
	hv.Invalidate()
}

func (hv *HomeView) usesStudentRail() bool {
	setting := hv.srv.Settings().FileInterface()
	return hv.studentMode && setting.NavigationLayout == settingsmodel.FileInterfaceNavigationLayoutLibrary
}

func (hv *HomeView) toggleStudentDrawer(section navpanel.NavSectionID) {
	if hv.CurrentView() == nil {
		return
	}
	if !hv.libraryActive && !hv.menuPanel.IsDrawerHidden && hv.sidebar.CurrentSection() == section {
		hv.menuPanel.IsDrawerHidden = true
	} else {
		hv.libraryActive = false
		hv.sidebar.ShowSection(section)
		hv.menuPanel.IsDrawerHidden = false
	}
	hv.Invalidate()
}

func (hv *HomeView) toggleConsole() {
	hv.consolePanel.ShowConsole = !hv.consolePanel.ShowConsole

	if !hv.consolePanel.ShowConsole {
		hv.lastYRatio = hv.yresizer.Ratio
		hv.yresizer.Ratio = 1.0
	} else {
		hv.yresizer.Ratio = hv.lastYRatio

	}
}

func (hv *HomeView) toggleChat() {
	projectDir := hv.srv.CurrentProjectDir()
	if projectDir == "" {
		return
	}

	intent := view.Intent{
		Target:      assistant.AgentChatViewID,
		ShowAsModal: false,
		RequireNew:  false,
	}
	hv.srv.RequestSwitch(intent)
}

func (hv *HomeView) update(gtx C) {
	// handle events and states update
	hv.sidebar.Update(gtx)
	showConsoleClicked, showChatClicked := hv.statusBar.Update(gtx)
	if showConsoleClicked {
		hv.toggleConsole()
	}
	if showChatClicked {
		hv.toggleChat()
	}
	if hv.chooseLibraryBtn.Clicked(gtx) {
		hv.menuPanel.OpenFolder()
	}

	// global key handler, without a focused target.
	for {
		e, ok := gtx.Event(
			key.Filter{Name: "D", Required: key.ModShortcut}, // toggle hide/show of drawer.
			key.Filter{Name: "K", Required: key.ModShortcut}, // toggle hide/show of console.
			key.Filter{Name: "L", Required: key.ModShortcut}, // toggle hide/show of chat.
		)
		if !ok {
			break
		}

		switch event := e.(type) {
		case key.Event:
			if event.State != key.Press {
				continue
			}

			if event.Name == "D" && event.Modifiers.Contain(key.ModShortcut) {
				if hv.usesStudentRail() {
					hv.toggleStudentDrawer(navpanel.NavSectionExplorer)
				} else {
					hv.menuPanel.IsDrawerHidden = !hv.menuPanel.IsDrawerHidden
				}
			}

			if event.Name == "K" && event.Modifiers.Contain(key.ModShortcut) {
				hv.toggleConsole()
			}

			if event.Name == "L" && event.Modifiers.Contain(key.ModShortcut) {
				hv.toggleChat()
			}
		}
	}

	if hv.accountClick.Clicked(gtx) {
		hv.RequestSwitch(view.Intent{
			Target: settings.SettingViewID,
			Params: map[string]any{
				"tabIdx": 3, // hardcoded tpix tab index in setting page.
			},
		})
	}
}

func (hv *HomeView) Layout(gtx C, th *theme.Theme, deco *widget.Decorations, title string) layout.Dimensions {
	hv.update(gtx)
	hv.sidebar.ShowLibrary = hv.studentMode && !hv.usesStudentRail()
	showLibraryPrompt := hv.studentMode && (hv.libraryActive && hv.libraryRoot == "" || hv.srv.Workspace().Current().Path == "" && hv.CurrentView() == nil)
	if showLibraryPrompt {
		hv.menuPanel.IsDrawerHidden = true
	}

	dims := layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Start,
	}.Layout(gtx,
		// layout.Rigid(func(gtx C) D {
		// 	d := decoration.Decorations(th, deco, ^system.Action(0), title)
		// 	d.Background = th.Bg2
		// 	d.Foreground = th.Fg
		// 	d.Title.Color = th.Fg
		// 	return d.Layout(gtx)
		// }),
		layout.Flexed(1, func(gtx C) D {
			// Store window layout metrics for native webview positioning.
			hv.srv.WindowContentWidth = gtx.Constraints.Max.X

			if hv.resizer == (widgets.Resize{}) {
				hv.resizer.Axis = layout.Horizontal
				hv.resizer.Ratio = float32(gtx.Dp(unit.Dp(280))) / float32(gtx.Constraints.Max.X)
				hv.lastResizeWidth = gtx.Constraints.Max.X
				hv.lastResizeRatio = hv.resizer.Ratio
			}

			if hv.lastResizeWidth != gtx.Constraints.Max.X {
				hv.resizer.Ratio = (float32(hv.lastResizeWidth) * hv.lastResizeRatio) / float32(gtx.Constraints.Max.X)
				hv.lastResizeWidth = gtx.Constraints.Max.X
				hv.lastResizeRatio = hv.resizer.Ratio
			} else if hv.lastResizeRatio != hv.resizer.Ratio {
				hv.lastResizeWidth = gtx.Constraints.Max.X
				hv.lastResizeRatio = hv.resizer.Ratio
			}

			layoutContent := func(gtx C) D {
				if hv.studentMode && hv.libraryActive || hv.menuPanel.IsDrawerHidden {
					return hv.layoutMain(gtx, th)
				}

				return hv.resizer.Layout(gtx,
					func(gtx C) D {
						return navpanel.NaviDrawerStyle{
							NavDrawer: hv.sidebar,
							Bg:        th.Bg2,
						}.Layout(gtx, th)
					},
					func(gtx C) D {
						return hv.layoutMain(gtx, th)
					},
					func(gtx C) D {
						if hv.bar == nil {
							hv.bar = widgets.NewResizeBar(layout.Vertical)
						}
						return hv.bar.Layout(gtx, th)
					},
				)
			}

			if !hv.usesStudentRail() {
				return layoutContent(gtx)
			}

			drawerVisible := !hv.libraryActive && !hv.menuPanel.IsDrawerHidden
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return hv.menuPanel.LayoutRail(gtx, th, hv.libraryActive || showLibraryPrompt, drawerVisible, hv.sidebar.CurrentSection())
				}),
				layout.Flexed(1, layoutContent),
			)
		}),
		layout.Rigid(func(gtx C) D {
			rect := clip.Rect{Max: gtx.Constraints.Max}
			paint.FillShape(gtx.Ops, th.Bg2, rect.Op())
			showViewStatus := !(hv.studentMode && hv.libraryActive && hv.libraryRoot != "")
			return layout.Flex{
				Gap:     gtx.Dp(unit.Dp(4)),
				Spacing: layout.SpaceBetween,
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return hv.menuPanel.Layout(gtx, th, hv.studentMode)
				}),
				layout.Flexed(1, func(gtx C) D {
					return hv.statusBar.Layout(gtx, th, showViewStatus)

				}),
			)
		}),
	)

	modalIter := hv.ModalViews()

	var allModals []*view.ModalView
	for modal := range modalIter {
		modal.Halted = true
		modal.MaxWidth = unit.Dp(960)
		modal.MaxHeight = 0.8
		modal.Radius = unit.Dp(8)
		modal.Padding = layout.Inset{
			Top:    unit.Dp(24),
			Bottom: unit.Dp(24),
			Left:   unit.Dp(20),
			Right:  unit.Dp(20),
		}

		allModals = append(allModals, modal)

	}

	for i, modal := range allModals {
		modal.ShowUp(gtx)

		if i == len(allModals)-1 {
			modal.Halted = false
		}

		// closing modal view
		if modal.IsClosed(gtx) {
			// should be the top most view.
			hv.FinishModalView()
			gtx.Execute(op.InvalidateCmd{})
		} else {
			modal.Layout(gtx, th)
		}

	}

	return dims
}

func (hv *HomeView) layoutMain(gtx C, th *theme.Theme) D {
	// draw the background
	gtx.Constraints.Min = gtx.Constraints.Max
	rect := clip.Rect{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, th.Bg, rect.Op())
	if hv.studentMode && (hv.libraryActive && hv.libraryRoot == "" || hv.srv.Workspace().Current().Path == "" && hv.CurrentView() == nil) {
		hv.srv.ViewAreaTopOffset = 0
		return hv.layoutLibraryPrompt(gtx, th)
	}

	if hv.libraryRequested {
		hv.libraryActive = true
		hv.menuPanel.IsDrawerHidden = true
		if filepath.Clean(hv.srv.Workspace().Current().Path) == filepath.Clean(hv.libraryRoot) {
			hv.libraryRequested = false
		}
	}
	if hv.studentMode && hv.libraryActive && hv.libraryRoot != "" {
		workspace := hv.srv.Workspace().Current()
		restorePreferred := filepath.Clean(workspace.Path) == filepath.Clean(hv.libraryRoot)
		preferred := ""
		if restorePreferred {
			preferred = workspace.LibraryPath
		}
		hv.library.SetWorkspace(hv.libraryRoot, preferred, restorePreferred)
		hv.srv.ViewAreaTopOffset = 0
		return hv.library.Layout(gtx, th)
	}

	rightPanelH := gtx.Constraints.Max.Y

	return layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		// horizontal navbar
		layout.Rigid(func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Spacing:   layout.SpaceBetween,
				Alignment: layout.Middle,
				Gap:       gtx.Dp(unit.Dp(4)),
			}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					return hv.tabbar.Layout(gtx, th)
				}),

				layout.Rigid(func(gtx C) D {
					return hv.layoutAccountInfo(gtx, th)
				}),
			)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.Spacer{Height: unit.Dp(1)}.Layout(gtx)
		}),

		layout.Flexed(1, func(gtx C) D {
			// Top offset = tabbar + spacer = rightPanelH - this child's height.
			hv.srv.ViewAreaTopOffset = rightPanelH - gtx.Constraints.Max.Y
			if !hv.consolePanel.ShowConsole {
				return hv.layoutView(gtx, th)
			}

			return hv.yresizer.Layout(gtx,
				func(gtx C) D {
					return hv.layoutView(gtx, th)
				},
				func(gtx C) D {
					return hv.consolePanel.Layout(gtx, th)
				},
				func(gtx C) D {
					if hv.hbar == nil {
						hv.hbar = widgets.NewResizeBar(layout.Horizontal)
					}

					return hv.hbar.Layout(gtx, th)
				},
			)
		}),
	)

}

func (hv *HomeView) layoutLibraryPrompt(gtx C, th *theme.Theme) D {
	return layout.Center.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min = image.Point{}
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(unit.Dp(480)))
		return widget.Border{Color: misc.WithAlpha(th.Fg, 0x30), Width: unit.Dp(1), CornerRadius: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
			return layout.Inset{Top: unit.Dp(30), Bottom: unit.Dp(30), Left: unit.Dp(34), Right: unit.Dp(34)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return libraryPromptIcon.Layout(gtx, th.ContrastBg, th.TextSize*3)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
					layout.Rigid(func(gtx C) D {
						label := material.H4(th.Theme, i18n.Translate("Choose your Library folder"))
						label.Font.Weight = font.SemiBold
						label.Alignment = text.Middle
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx C) D {
						label := material.Body1(th.Theme, i18n.Translate("Select the folder that contains your courses, notebooks, and documents. Typstify keeps it as a normal folder."))
						label.Color = misc.WithAlpha(th.Fg, 0xb0)
						label.Alignment = text.Middle
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(22)}.Layout),
					layout.Rigid(material.Button(th.Theme, &hv.chooseLibraryBtn, i18n.Translate("Choose folder")).Layout),
				)
			})
		})
	})
}

func (hv *HomeView) layoutView(gtx C, th *theme.Theme) D {
	cv := hv.CurrentView()
	if cv == nil {
		return hv.welcome.Layout(gtx, th)
	}

	pv, ok := cv.(previewable)
	if ok {
		pv.SetPreviewer(hv.previewer)
	}

	showPreview := ok && pv.IsVisible()

	if !showPreview {
		hv.previewVisible = false
		return cv.Layout(gtx, th)
	}

	// Preview is visible.
	if hv.previewResizer == nil {
		hv.previewResizer = &widgets.Resize{Axis: layout.Horizontal}
	}
	if !hv.previewVisible {
		hv.previewResizer.Ratio = editorRatioForPreviewWidth(hv.srv.Settings().Editor().PreviewWidth)
		hv.previewVisible = true
	}

	return hv.previewResizer.Layout(gtx,
		func(gtx C) D {
			return cv.Layout(gtx, th)
		},
		func(gtx C) D {
			return pv.LayoutPreview(gtx, th)
		},
		func(gtx C) D {
			if hv.previewBar == nil {
				hv.previewBar = widgets.NewResizeBar(layout.Vertical)
			}
			return hv.previewBar.Layout(gtx, th)
		},
	)
}

func editorRatioForPreviewWidth(previewWidth int) float32 {
	if previewWidth < 20 || previewWidth > 60 {
		previewWidth = 30
	}
	return 1 - float32(previewWidth)/100
}

func (hv *HomeView) layoutAccountInfo(gtx C, th *theme.Theme) D {
	authInfo := hv.srv.TpixSessionService().Session()
	userAuthed := hv.srv.TpixSessionService().Authenticated()

	return hv.accountClick.Layout(gtx, func(gtx C) D {
		paintColor := th.Fg
		if hv.accountClick.Hovered() {
			paintColor = th.ContrastBg
		}

		return layout.Inset{
			Top:    unit.Dp(2),
			Bottom: unit.Dp(2),
			Left:   unit.Dp(4),
			Right:  unit.Dp(12),
		}.Layout(gtx, func(gtx C) D {
			return layout.Flex{
				Axis:      layout.Horizontal,
				Alignment: layout.Middle,
				Gap:       gtx.Dp(unit.Dp(4)),
			}.Layout(gtx,
				layout.Rigid(func(gtx C) D {
					return userIcon.Layout(gtx, paintColor, th.TextSize*0.8)
				}),
				layout.Rigid(func(gtx C) D {
					if userAuthed {
						name := authInfo.Username
						// if user is not subscribed, show the state and prompt he/her to upgrade.
						if !authInfo.Subscribed {
							name = fmt.Sprintf("%s (unsubscribed)", name)
						}
						lb := material.Label(th.Theme, th.TextSize, name)
						lb.Color = paintColor
						return lb.Layout(gtx)
					}

					lb := material.Label(th.Theme, th.TextSize, i18n.Translate("Sign In"))
					lb.Color = paintColor
					return lb.Layout(gtx)

				}),
			)
		})
	})

}

func (hv *HomeView) OnClose() {
	hv.sidebar.Close()
	hv.library.Close()
	hv.srv.EventBus().Unsubscribe(hv)
	if hv.previewer != nil {
		hv.previewer.Destroy()
	}
}

func newHome(window *app.Window, srv *service.ServiceFacade) *HomeView {
	vm := view.DefaultViewManager(window)
	fileInterface := srv.Settings().FileInterface()
	hv := &HomeView{
		ViewManager:  vm,
		srv:          srv,
		consolePanel: console.NewConsolePanel(srv.Console()),
		yresizer:     &widgets.Resize{Axis: layout.Vertical, Ratio: 1.0},
		lastYRatio:   0.7,
		previewer:    preview.NewPreviewer(srv),
		studentMode:  fileInterface.Mode == settingsmodel.FileInterfaceModeStudent,
		libraryRoot:  fileInterface.LibraryRoot,
	}
	hv.tabbar = navpanel.NewTabbar(hv, nil)
	hv.sidebar = navpanel.NewNavDrawer(hv, srv)
	hv.sidebar.OnLibrary = hv.showLibrary
	hv.statusBar = statusbar.NewStatusBar(srv, hv)
	hv.menuPanel = navpanel.NewMenuPanel(hv, srv)
	hv.menuPanel.OnLibrary = hv.showLibrary
	hv.menuPanel.OnExplorer = func() { hv.toggleStudentDrawer(navpanel.NavSectionExplorer) }
	hv.menuPanel.OnOutline = func() { hv.toggleStudentDrawer(navpanel.NavSectionOutline) }
	hv.menuPanel.OnAssistant = func() { hv.toggleStudentDrawer(navpanel.NavSectionAssistant) }
	hv.library = navpanel.NewLibrary(srv, hv)
	hv.library.OnFileOpen = func(scope string, showExplorer bool) {
		hv.sidebar.ShowFileTree(scope)
		hv.menuPanel.IsDrawerHidden = !showExplorer
	}
	hv.welcome = WelcomeView{vm: hv, srv: srv}

	srv.EventBus().Subscribe(hv, "home.settings", `settings\.updated`, func(topic string, data interface{}) {
		if editorSetting, ok := data.(*settingsmodel.EditorSettings); ok {
			if hv.previewResizer != nil {
				hv.previewResizer.Ratio = editorRatioForPreviewWidth(editorSetting.PreviewWidth)
			}
			hv.Invalidate()
			return
		}

		setting, ok := data.(*settingsmodel.FileInterfaceSettings)
		if !ok {
			return
		}
		studentMode := setting.Mode == settingsmodel.FileInterfaceModeStudent
		if studentMode && !hv.studentMode {
			hv.studentMode = true
			hv.showLibrary()
		} else if !studentMode {
			hv.studentMode = false
			hv.libraryActive = false
		} else {
			hv.libraryRoot = setting.LibraryRoot
		}
		hv.Invalidate()
	})
	srv.EventBus().Subscribe(hv, "home.workspace", `project\.(switched|create)$`, func(topic string, data interface{}) {
		if !hv.studentMode {
			return
		}
		if topic == bus.TopicProjectCreate {
			created, ok := data.(bus.ProjectCreatedEvent)
			if !ok {
				return
			}
			if created.SwitchWorkspace && hv.libraryRoot == "" {
				hv.libraryRoot = created.Path
			}
			hv.libraryRequested = false
			hv.libraryActive = false
			hv.sidebar.ShowSection(navpanel.NavSectionExplorer)
			hv.menuPanel.IsDrawerHidden = false
			hv.Invalidate()
			return
		}

		root, ok := data.(string)
		if ok {
			setting := srv.Settings().FileInterface()
			if setting.LibraryRoot != root {
				setting.LibraryRoot = root
				if err := setting.Save(); err != nil {
					srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: err.Error(), Level: 1})
				}
			}
			hv.libraryRoot = root
			hv.libraryRequested = true
			hv.menuPanel.IsDrawerHidden = true
		}
		hv.Invalidate()
	})
	if hv.studentMode && hv.libraryRoot != "" {
		srv.EventBus().Emit(bus.TopicProjectSwitched, hv.libraryRoot)
	}

	return hv
}
