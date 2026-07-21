package navpanel

import (
	"image"
	"log"
	"path/filepath"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"

	// "golang.org/x/exp/shiny/materialdesign/icons"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	settingsmodel "looz.ws/typstify/service/settings"
	"looz.ws/typstify/ui/dialog"
	"looz.ws/typstify/ui/pkgmgmt"
	"looz.ws/typstify/ui/settings"
	wg "looz.ws/typstify/widgets"
	"looz.ws/typstify/widgets/icons"
)

var (
	openFolder        = icons.NewSvgIcon(icons.FolderOpen)
	newFolder         = icons.NewSvgIcon(icons.FolderPlus)
	historyIcon       = icons.NewSvgIcon(icons.History)
	pkgManagerIcon    = icons.NewSvgIcon(icons.PackageOpen)
	settingsIcon      = icons.NewSvgIcon(icons.Cog)
	panelHideIcon     = icons.NewSvgIcon(icons.PanelLeftClose)
	panelShowIcon     = icons.NewSvgIcon(icons.PanelRightClose)
	libraryIcon       = icons.NewSvgIcon(icons.Home)
	railExplorerIcon  = icons.NewSvgIcon(icons.FolderTree)
	railOutlineIcon   = icons.NewSvgIcon(icons.TableOfContents)
	railAssistantIcon = icons.NewSvgIcon(icons.MessagesSquare)
)

type MenuPanel struct {
	openDirBtn        widget.Clickable
	openDirTip        wg.TipArea
	openPkgManagerBtn widget.Clickable
	openPkgManagerTip wg.TipArea
	newProjectBtn     widget.Clickable
	newProjectTip     wg.TipArea
	openSettingBtn    widget.Clickable
	openSettingTip    wg.TipArea
	hideDrawerBtn     widget.Clickable
	hideDrawerTip     wg.TipArea
	openLibraryBtn    widget.Clickable
	openLibraryTip    wg.TipArea
	openExplorerBtn   widget.Clickable
	openExplorerTip   wg.TipArea
	openOutlineBtn    widget.Clickable
	openOutlineTip    wg.TipArea
	openAssistantBtn  widget.Clickable
	openAssistantTip  wg.TipArea

	IsDrawerHidden bool
	OnLibrary      func()
	OnExplorer     func()
	OnOutline      func()
	OnAssistant    func()
	vm             view.ViewManager
	srv            *service.ServiceFacade
}

func (cp *MenuPanel) Layout(gtx C, th *theme.Theme, studentMode bool) D {
	if studentMode && cp.srv.Settings().FileInterface().NavigationLayout == settingsmodel.FileInterfaceNavigationLayoutLibrary {
		return D{}
	}
	cp.update(gtx)

	return layout.Inset{
		Left:   unit.Dp(8),
		Top:    unit.Dp(4),
		Bottom: unit.Dp(4),
	}.Layout(gtx, func(gtx C) D {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.hideDrawerTip, i18n.Translate("Hide Explorer"))
				return btn.Layout(gtx, func(gtx C) D {
					icon := panelHideIcon
					if cp.IsDrawerHidden {
						icon = panelShowIcon
					}
					return cp.layoutBtn(gtx, th, &cp.hideDrawerBtn, icon)
				})
			}),
		}
		if studentMode {
			children = append(children, layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.openLibraryTip, i18n.Translate("Library"))
				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.openLibraryBtn, libraryIcon)
				})
			}))
		}
		children = append(children,
			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.openDirTip, i18n.Translate("Open Folder"))
				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.openDirBtn, openFolder)
				})
			}),
			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.newProjectTip, i18n.Translate("New Project"))
				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.newProjectBtn, newFolder)
				})
			}),
			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.openPkgManagerTip, i18n.Translate("Typst Package Center"))
				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.openPkgManagerBtn, pkgManagerIcon)
				})
			}),
			layout.Rigid(func(gtx C) D {
				btn := wg.TipIconButton(th, &cp.openSettingTip, i18n.Translate("Settings"))
				return btn.Layout(gtx, func(gtx C) D {
					return cp.layoutBtn(gtx, th, &cp.openSettingBtn, settingsIcon)
				})
			}),
		)

		return layout.Flex{
			Axis:    layout.Horizontal,
			Spacing: layout.SpaceEnd,
			Gap:     gtx.Dp(unit.Dp(16)),
		}.Layout(gtx, children...)
	})
}

func (cp *MenuPanel) LayoutRail(gtx C, th *theme.Theme, libraryActive, drawerVisible bool, section NavSectionID) D {
	cp.update(gtx)

	width := gtx.Dp(unit.Dp(46))
	gtx.Constraints.Min.X = width
	gtx.Constraints.Max.X = width
	gtx.Constraints.Min.Y = gtx.Constraints.Max.Y

	return layout.Background{}.Layout(gtx,
		func(gtx C) D {
			dims := D{Size: gtx.Constraints.Max}
			paint.FillShape(gtx.Ops, th.Bg2, clip.Rect{Max: dims.Size}.Op())
			return dims
		},
		func(gtx C) D {
			return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.openLibraryBtn, &cp.openLibraryTip, i18n.Translate("Library"), libraryIcon, libraryActive)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.openExplorerBtn, &cp.openExplorerTip, i18n.Translate("Files"), railExplorerIcon, drawerVisible && section == NavSectionExplorer)
					}),
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.openOutlineBtn, &cp.openOutlineTip, i18n.Translate("Outline"), railOutlineIcon, drawerVisible && section == NavSectionOutline)
					}),
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.openAssistantBtn, &cp.openAssistantTip, i18n.Translate("Assistant Sessions"), railAssistantIcon, drawerVisible && section == NavSectionAssistant)
					}),
					layout.Flexed(1, func(gtx C) D { return D{} }),
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.openDirBtn, &cp.openDirTip, i18n.Translate("Open Folder"), openFolder, false)
					}),
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.newProjectBtn, &cp.newProjectTip, i18n.Translate("New Project"), newFolder, false)
					}),
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.openPkgManagerBtn, &cp.openPkgManagerTip, i18n.Translate("Typst Package Center"), pkgManagerIcon, false)
					}),
					layout.Rigid(func(gtx C) D {
						return cp.layoutRailButton(gtx, th, &cp.openSettingBtn, &cp.openSettingTip, i18n.Translate("Settings"), settingsIcon, false)
					}),
				)
			})
		},
	)
}

func (cp *MenuPanel) layoutRailButton(gtx C, th *theme.Theme, btn *widget.Clickable, tip *wg.TipArea, label string, icon *icons.SvgIcon, selected bool) D {
	return layout.Inset{Top: unit.Dp(2), Bottom: unit.Dp(2), Left: unit.Dp(6), Right: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
		return wg.TipIconButton(th, tip, label).Layout(gtx, func(gtx C) D {
			return btn.Layout(gtx, func(gtx C) D {
				size := gtx.Dp(unit.Dp(34))
				gtx.Constraints = layout.Exact(image.Pt(size, size))
				background := th.Bg2
				color := th.Fg
				if selected {
					background = misc.WithAlpha(th.ContrastBg, th.SelectedAlpha)
					color = th.ContrastBg
				} else if btn.Hovered() {
					background = misc.WithAlpha(th.ContrastBg, th.HoverAlpha)
				}
				return layout.Background{}.Layout(gtx,
					func(gtx C) D {
						dims := D{Size: gtx.Constraints.Max}
						paint.FillShape(gtx.Ops, background, clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(unit.Dp(6))).Op(gtx.Ops))
						return dims
					},
					func(gtx C) D {
						return layout.Center.Layout(gtx, func(gtx C) D { return icon.Layout(gtx, color, th.TextSize) })
					},
				)
			})
		})
	})
}

func (cp *MenuPanel) layoutBtn(gtx C, th *theme.Theme, btn *widget.Clickable, icon *icons.SvgIcon) D {
	return btn.Layout(gtx, func(gtx C) D {
		return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx C) D {
			return icon.Layout(gtx, th.Fg, th.TextSize)
		})
	})
}

func (cp *MenuPanel) update(gtx C) {
	cp.openDirTip.Direction = layout.E
	cp.newProjectTip.Direction = layout.E
	cp.openPkgManagerTip.Direction = layout.E
	cp.openSettingTip.Direction = layout.E
	cp.hideDrawerTip.Direction = layout.E
	cp.openLibraryTip.Direction = layout.E
	cp.openExplorerTip.Direction = layout.E
	cp.openOutlineTip.Direction = layout.E
	cp.openAssistantTip.Direction = layout.E

	if cp.openLibraryBtn.Clicked(gtx) && cp.OnLibrary != nil {
		cp.OnLibrary()
	}
	if cp.openExplorerBtn.Clicked(gtx) && cp.OnExplorer != nil {
		cp.OnExplorer()
	}
	if cp.openOutlineBtn.Clicked(gtx) && cp.OnOutline != nil {
		cp.OnOutline()
	}
	if cp.openAssistantBtn.Clicked(gtx) && cp.OnAssistant != nil {
		cp.OnAssistant()
	}

	if cp.openSettingBtn.Clicked(gtx) {
		cp.vm.RequestSwitch(view.Intent{
			Target:     settings.SettingViewID,
			RequireNew: true,
		})
	}

	if cp.newProjectBtn.Clicked(gtx) {
		cp.vm.RequestSwitch(view.Intent{
			Target:      dialog.CreateProjectDialogViewID,
			ShowAsModal: true,
		})
	}

	if cp.openDirBtn.Clicked(gtx) {
		cp.OpenFolder()
	}

	if cp.openPkgManagerBtn.Clicked(gtx) {
		cp.vm.RequestSwitch(view.Intent{
			Target:     pkgmgmt.PkgListViewID,
			RequireNew: true,
		})
	}

	if cp.hideDrawerBtn.Clicked(gtx) {
		cp.IsDrawerHidden = !cp.IsDrawerHidden
	}
}

func (cp *MenuPanel) OpenFolder() {
	go func() {
		projectDir, err := cp.srv.FileChooser().ChooseFolder()
		if err != nil {
			log.Println("failed to choose folder: ", projectDir, err)
			return
		}
		if isFile(projectDir) {
			projectDir = filepath.Dir(projectDir)
		}

		log.Println("choosed folder: ", projectDir)
		cp.srv.EventBus().Emit(bus.TopicProjectSwitched, projectDir)
	}()
}

func NewMenuPanel(vm view.ViewManager, srv *service.ServiceFacade) *MenuPanel {
	return &MenuPanel{
		vm:  vm,
		srv: srv,
	}
}
