package navpanel

import (
	"image"
	"image/color"
	"path/filepath"
	"strings"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/explorer"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	settingsmodel "looz.ws/typstify/service/settings"
	"looz.ws/typstify/ui/dialog"
	"looz.ws/typstify/widgets"
	"looz.ws/typstify/widgets/filetree"
	"looz.ws/typstify/widgets/icons"
)

var (
	libraryUpIcon     = icons.NewSvgIcon(icons.ArrowUp)
	libraryGridIcon   = icons.NewSvgIcon(icons.Columns2)
	libraryListIcon   = icons.NewSvgIcon(icons.List)
	libraryFilterIcon = icons.NewSvgIcon(icons.ListFilter)
	libraryCheckIcon  = icons.NewSvgIcon(icons.SquareCheck)
	libraryNewIcon    = icons.NewSvgIcon(icons.FilePlus)
	libraryCrumbIcon  = icons.NewSvgIcon(icons.ChevronRight)
)

type libraryFilter string

const (
	libraryFilterAll       libraryFilter = "all"
	libraryFilterTypst     libraryFilter = "typst"
	libraryFilterNotebooks libraryFilter = "notebooks"
	libraryFilterDocuments libraryFilter = "documents"
	libraryFilterPDFs      libraryFilter = "pdfs"
	libraryFilterImages    libraryFilter = "images"
	libraryFilterOther     libraryFilter = "other"
)

type Library struct {
	srv *service.ServiceFacade
	vm  view.ViewManager

	browser         *filetree.FocusedBrowser
	tree            *filetree.TreeView
	entries         []filetree.FocusedEntry
	root            string
	preferredLoaded bool
	lastErr         error

	list         widget.List
	upBtn        widget.Clickable
	newBtn       widget.Clickable
	gridBtn      widget.Clickable
	listBtn      widget.Clickable
	filterBtn    widget.Clickable
	crumbBtns    map[string]*widget.Clickable
	newPopup     *widgets.Popup
	filterPopup  *widgets.Popup
	pendingEnter string
	pendingNew   string
	filter       libraryFilter

	OnFileOpen func(scope string, showExplorer bool)
}

func NewLibrary(srv *service.ServiceFacade, vm view.ViewManager) *Library {
	return &Library{
		srv:         srv,
		vm:          vm,
		crumbBtns:   make(map[string]*widget.Clickable),
		newPopup:    &widgets.Popup{Width: unit.Dp(180), Direction: layout.S},
		filterPopup: &widgets.Popup{Width: unit.Dp(220), Direction: layout.S},
		list:        widget.List{List: layout.List{Axis: layout.Vertical}},
		filter:      libraryFilterTypst,
	}
}

func (l *Library) SetWorkspace(root, preferred string, restorePreferred bool) {
	root = filepath.Clean(root)
	if root == "." || root == "" {
		return
	}
	if l.browser != nil && l.root == root {
		if restorePreferred && !l.preferredLoaded {
			l.preferredLoaded = true
			if err := l.reset(root, preferred); err != nil {
				l.lastErr = err
			} else {
				l.lastErr = nil
			}
		}
		return
	}

	l.preferredLoaded = restorePreferred
	if err := l.reset(root, preferred); err != nil {
		l.lastErr = err
	} else {
		l.lastErr = nil
	}
}

func (l *Library) reset(root, preferred string) error {
	browser, err := filetree.NewFocusedBrowser(root, preferred)
	if err != nil {
		return err
	}
	l.root = browser.Root()
	l.browser = browser
	l.resetTree()
	return l.refreshEntries()
}

func (l *Library) resetTree() {
	if l.tree != nil {
		l.tree.Close()
	}
	l.tree = filetree.NewTreeView(l.browser.Node())
	configureFileTree(l.tree, l.srv, l.vm)
}

func (l *Library) refreshEntries() error {
	if l.browser == nil {
		return nil
	}
	current := l.browser.Current()
	entries, err := l.browser.Entries()
	if err != nil {
		return err
	}
	l.entries = entries
	if current != l.browser.Current() || l.tree == nil || l.tree.Root() != l.browser.Current() {
		l.resetTree()
		l.srv.Workspace().SaveLibraryPath(l.browser.Current())
	}
	return nil
}

func (l *Library) enter(path string) {
	if err := l.browser.Enter(path); err != nil {
		l.lastErr = err
		return
	}
	l.srv.Workspace().SaveLibraryPath(l.browser.Current())
	l.resetTree()
	l.lastErr = l.refreshEntries()
}

func (l *Library) update(gtx C) {
	if l.browser == nil || l.tree == nil {
		return
	}

	if l.pendingEnter != "" {
		path := l.pendingEnter
		l.pendingEnter = ""
		l.enter(path)
	}
	if l.upBtn.Clicked(gtx) {
		if l.browser.Current() != l.browser.Root() {
			l.enter(filepath.Dir(l.browser.Current()))
		}
	}
	for _, crumb := range l.browser.Breadcrumbs() {
		if btn := l.crumbBtns[crumb.Path]; btn != nil && btn.Clicked(gtx) {
			l.enter(crumb.Path)
			break
		}
	}
	if l.newBtn.Clicked(gtx) {
		l.newPopup.SetOpen()
	}
	if l.gridBtn.Clicked(gtx) {
		l.saveView(settingsmodel.FileInterfaceLibraryViewGrid)
	}
	if l.listBtn.Clicked(gtx) {
		l.saveView(settingsmodel.FileInterfaceLibraryViewList)
	}
	if l.filterBtn.Clicked(gtx) {
		l.filterPopup.SetOpen()
	}

	switch l.pendingNew {
	case "file":
		l.lastErr = l.tree.CreateChild(gtx, l.browser.Node(), explorer.FileNode)
	case "folder":
		l.lastErr = l.tree.CreateChild(gtx, l.browser.Node(), explorer.FolderNode)
	case "project":
		l.vm.RequestSwitch(view.Intent{
			Target:      dialog.CreateProjectDialogViewID,
			ShowAsModal: true,
			Params: map[string]any{
				dialog.ProjectDirParam:      l.browser.Current(),
				dialog.FixedProjectDirParam: true,
			},
		})
	}
	if l.pendingNew != "" {
		l.pendingNew = ""
		if err := l.refreshEntries(); err != nil {
			l.lastErr = err
		}
	}

	if l.tree.Update(gtx) {
		l.lastErr = l.refreshEntries()
	}
}

func (l *Library) saveView(value settingsmodel.FileInterfaceLibraryView) {
	setting := l.srv.Settings().FileInterface()
	setting.LibraryView = value
	if err := setting.Save(); err != nil {
		l.lastErr = err
	}
}

func (l *Library) Layout(gtx C, th *theme.Theme) D {
	l.update(gtx)
	if l.browser == nil || l.tree == nil {
		return D{}
	}

	macro := op.Record(gtx.Ops)
	dims := l.layout(gtx, th)
	call := macro.Stop()

	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, l.tree)
	call.Add(gtx.Ops)
	l.tree.LayoutContextMenu(gtx, th)
	return dims
}

func (l *Library) layout(gtx C, th *theme.Theme) D {
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D { return l.layoutHeader(gtx, th) }),
		layout.Rigid(func(gtx C) D {
			if l.lastErr == nil {
				return D{}
			}
			return layout.Inset{Left: unit.Dp(24), Right: unit.Dp(24), Bottom: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
				return misc.LayoutErrorLabel(gtx, th, l.lastErr)
			})
		}),
		layout.Flexed(1, func(gtx C) D {
			return layout.Inset{Left: unit.Dp(24), Right: unit.Dp(24), Bottom: unit.Dp(20)}.Layout(gtx, func(gtx C) D {
				entries := l.visibleEntries()
				if len(entries) == 0 {
					return layout.Center.Layout(gtx, func(gtx C) D {
						message := i18n.Translate("No items match this filter.")
						if len(l.entries) == 0 {
							message = i18n.Translate("This folder is empty. Use New to create a file, folder, or project.")
						}
						label := material.Body1(th.Theme, message)
						label.Color = misc.WithAlpha(th.Fg, 0xb6)
						return label.Layout(gtx)
					})
				}
				if l.srv.Settings().FileInterface().LibraryView == settingsmodel.FileInterfaceLibraryViewList {
					return l.layoutList(gtx, th, entries)
				}
				return l.layoutGrid(gtx, th, entries)
			})
		}),
	)
}

func (l *Library) layoutHeader(gtx C, th *theme.Theme) D {
	return layout.Inset{Top: unit.Dp(20), Bottom: unit.Dp(18), Left: unit.Dp(24), Right: unit.Dp(24)}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return l.upBtn.Layout(gtx, func(gtx C) D {
							return layout.Inset{Right: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
								color := th.Fg
								if l.browser.Current() == l.browser.Root() {
									color = misc.WithAlpha(th.Fg, 0x60)
								}
								gtx.Constraints = layout.Exact(image.Pt(gtx.Dp(unit.Dp(36)), gtx.Dp(unit.Dp(36))))
								return widget.Border{Color: misc.WithAlpha(th.Fg, 0x38), Width: unit.Dp(1), CornerRadius: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
									return layout.Center.Layout(gtx, func(gtx C) D {
										return libraryUpIcon.Layout(gtx, color, th.TextSize*1.2)
									})
								})
							})
						})
					}),
					layout.Flexed(1, func(gtx C) D { return l.layoutBreadcrumbs(gtx, th) }),
				)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(18)}.Layout),
			layout.Rigid(func(gtx C) D {
				return layout.Flex{Alignment: layout.Middle, Spacing: layout.SpaceBetween}.Layout(gtx,
					layout.Flexed(1, func(gtx C) D {
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								title := material.H4(th.Theme, filepath.Base(l.browser.Current()))
								title.Font.Weight = font.SemiBold
								return title.Layout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								label := material.Caption(th.Theme, i18n.Translate("%d items", len(l.visibleEntries())))
								label.Color = misc.WithAlpha(th.Fg, 0xb6)
								return label.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(func(gtx C) D { return l.layoutActions(gtx, th) }),
				)
			}),
		)
	})
}

func (l *Library) layoutBreadcrumbs(gtx C, th *theme.Theme) D {
	children := make([]layout.FlexChild, 0)
	crumbs := l.browser.Breadcrumbs()
	for i, crumb := range crumbs {
		crumb := crumb
		current := i == len(crumbs)-1
		btn := l.crumbBtns[crumb.Path]
		if btn == nil {
			btn = &widget.Clickable{}
			l.crumbBtns[crumb.Path] = btn
		}
		if i > 0 {
			children = append(children, layout.Rigid(func(gtx C) D {
				return layout.Inset{Left: unit.Dp(3), Right: unit.Dp(3)}.Layout(gtx, func(gtx C) D {
					return libraryCrumbIcon.Layout(gtx, misc.WithAlpha(th.Fg, 0x80), th.TextSize*0.9)
				})
			}))
		}
		children = append(children, layout.Rigid(func(gtx C) D {
			return btn.Layout(gtx, func(gtx C) D {
				return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
					label := material.Body1(th.Theme, crumb.Name)
					label.Color = th.ContrastBg
					if current {
						label.Color = th.Fg
						label.Font.Weight = font.SemiBold
					}
					return label.Layout(gtx)
				})
			})
		}))
	}
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
}

func (l *Library) layoutActions(gtx C, th *theme.Theme) D {
	viewSetting := l.srv.Settings().FileInterface().LibraryView
	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return l.filterPopup.Layout(gtx, th, func(gtx C) D {
				return l.filterBtn.Layout(gtx, func(gtx C) D {
					return widget.Border{Color: misc.WithAlpha(th.Fg, 0x40), Width: unit.Dp(1), CornerRadius: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
						return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(9), Right: unit.Dp(9)}.Layout(gtx, func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx C) D { return libraryFilterIcon.Layout(gtx, th.Fg, th.TextSize) }),
								layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
								layout.Rigid(material.Body2(th.Theme, l.filterLabel()).Layout),
							)
						})
					})
				})
			}, l.filterItems()...)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
		layout.Rigid(func(gtx C) D {
			return l.layoutViewButton(gtx, th, &l.gridBtn, libraryGridIcon, viewSetting == settingsmodel.FileInterfaceLibraryViewGrid)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Rigid(func(gtx C) D {
			return l.layoutViewButton(gtx, th, &l.listBtn, libraryListIcon, viewSetting == settingsmodel.FileInterfaceLibraryViewList)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
		layout.Rigid(func(gtx C) D {
			return l.newPopup.Layout(gtx, th, func(gtx C) D {
				return l.newBtn.Layout(gtx, func(gtx C) D {
					return widget.Border{Color: misc.WithAlpha(th.Fg, 0x40), Width: unit.Dp(1), CornerRadius: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
						return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
							return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx C) D { return libraryNewIcon.Layout(gtx, th.Fg, th.TextSize) }),
								layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
								layout.Rigid(material.Body2(th.Theme, i18n.Translate("New")).Layout),
							)
						})
					})
				})
			},
				libraryMenuItem{name: i18n.Translate("New File"), onClick: func() { l.pendingNew = "file" }},
				libraryMenuItem{name: i18n.Translate("New Folder"), onClick: func() { l.pendingNew = "folder" }},
				libraryMenuItem{name: i18n.Translate("New Project"), onClick: func() { l.pendingNew = "project" }},
			)
		}),
	)
}

func (l *Library) layoutViewButton(gtx C, th *theme.Theme, btn *widget.Clickable, icon *icons.SvgIcon, selected bool) D {
	return btn.Layout(gtx, func(gtx C) D {
		color := th.Fg
		background := th.Bg
		if selected {
			color = th.ContrastBg
			background = misc.WithAlpha(th.ContrastBg, th.SelectedAlpha)
		}
		return layout.Background{}.Layout(gtx,
			func(gtx C) D {
				dims := D{Size: image.Pt(gtx.Dp(unit.Dp(30)), gtx.Dp(unit.Dp(30)))}
				paint.FillShape(gtx.Ops, background, clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(unit.Dp(5))).Op(gtx.Ops))
				return dims
			},
			func(gtx C) D {
				return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D { return icon.Layout(gtx, color, th.TextSize) })
			},
		)
	})
}

func (l *Library) layoutGrid(gtx C, th *theme.Theme, entries []filetree.FocusedEntry) D {
	const gap = unit.Dp(14)
	cardWidth := gtx.Dp(unit.Dp(160))
	columns := max(1, (gtx.Constraints.Max.X+gtx.Dp(gap))/(cardWidth+gtx.Dp(gap)))
	rows := (len(entries) + columns - 1) / columns
	return material.List(th.Theme, &l.list).Layout(gtx, rows, func(gtx C, row int) D {
		children := make([]layout.FlexChild, 0, columns*2)
		for column := 0; column < columns; column++ {
			index := row*columns + column
			if column > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Width: gap}.Layout))
			}
			if index >= len(entries) {
				children = append(children, layout.Flexed(1, func(gtx C) D { return D{} }))
				continue
			}
			entry := entries[index]
			children = append(children, layout.Flexed(1, func(gtx C) D {
				return layout.Inset{Bottom: gap}.Layout(gtx, func(gtx C) D { return l.layoutCard(gtx, th, entry) })
			}))
		}
		return layout.Flex{}.Layout(gtx, children...)
	})
}

func (l *Library) layoutList(gtx C, th *theme.Theme, entries []filetree.FocusedEntry) D {
	return material.List(th.Theme, &l.list).Layout(gtx, len(entries), func(gtx C, index int) D {
		return layout.Inset{Bottom: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
			return l.layoutEntry(gtx, th, entries[index], func(gtx C, state *filetree.NodeState) D {
				return widget.Border{Color: misc.WithAlpha(th.Fg, 0x24), Width: unit.Dp(1), CornerRadius: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
					return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D { return l.layoutEntryIcon(gtx, th, entries[index], th.TextSize*1.4) }),
							layout.Rigid(layout.Spacer{Width: unit.Dp(12)}.Layout),
							layout.Flexed(1, func(gtx C) D { return l.layoutEntryName(gtx, th, state) }),
							layout.Rigid(func(gtx C) D {
								label := material.Caption(th.Theme, l.entryKind(entries[index]))
								label.Color = misc.WithAlpha(th.Fg, 0x90)
								return label.Layout(gtx)
							}),
						)
					})
				})
			})
		})
	})
}

func (l *Library) layoutCard(gtx C, th *theme.Theme, entry filetree.FocusedEntry) D {
	cardHeight := gtx.Dp(unit.Dp(196))
	gtx.Constraints.Min.Y = cardHeight
	gtx.Constraints.Max.Y = cardHeight
	return l.layoutEntry(gtx, th, entry, func(gtx C, state *filetree.NodeState) D {
		gtx.Constraints.Min.X = gtx.Constraints.Max.X
		return layout.Inset{Top: unit.Dp(12), Bottom: unit.Dp(12), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
			iconSize := th.TextSize * 4
			if entry.IsNotebook {
				iconSize = th.TextSize * 6.5
			} else if entry.Node.IsDir() {
				iconSize = th.TextSize * 4.5
			}
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx C) D {
					return layout.Center.Layout(gtx, func(gtx C) D { return l.layoutEntryIcon(gtx, th, entry, iconSize) })
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx C) D { return l.layoutEntryName(gtx, th, state) }),
			)
		})
	})
}

func (l *Library) layoutEntry(gtx C, th *theme.Theme, entry filetree.FocusedEntry, content func(gtx C, state *filetree.NodeState) D) D {
	state := l.tree.PrepareNode(entry.Node)
	state.Label.Radius = unit.Dp(10)
	state.Editable.Color = th.Fg
	state.Editable.TextSize = th.TextSize
	if state.Label.Update(gtx) {
		if entry.IsNotebook {
			entrypoint, err := filetree.NotebookEntrypoint(entry.Node.Path)
			if err != nil {
				l.pendingEnter = entry.Node.Path
			} else if node, err := explorer.NewFileTree(entrypoint); err != nil {
				l.lastErr = err
			} else {
				if l.OnFileOpen != nil {
					l.OnFileOpen(entry.Node.Path, true)
				}
				l.tree.OnSelect(node)
			}
		} else {
			if !entry.Node.IsDir() && l.OnFileOpen != nil {
				scope := filepath.Dir(entry.Node.Path)
				showExplorer := false
				if resolved, kind, err := filetree.ResolveEditorScope(l.root, entry.Node.Path); err == nil {
					scope = resolved
					showExplorer = kind == filetree.EditorScopeProject
				}
				l.OnFileOpen(scope, showExplorer)
			}
			l.tree.OnSelect(entry.Node)
		}
		if entry.Node.IsDir() && !entry.IsNotebook {
			l.pendingEnter = entry.Node.Path
		}
		gtx.Execute(op.InvalidateCmd{})
	}
	flat := filetree.FlatNode{Node: entry.Node, State: state}
	if err := flat.Update(gtx, l.tree); err != nil {
		l.lastErr = err
	}

	return l.layoutInteractiveEntry(gtx, th, entry, state, content)
}

func (l *Library) layoutInteractiveEntry(gtx C, th *theme.Theme, entry filetree.FocusedEntry, state *filetree.NodeState, content func(gtx C, state *filetree.NodeState) D) D {
	macro := op.Record(gtx.Ops)
	dims := state.Label.Layout(gtx, th, func(gtx C, _ color.NRGBA) D {
		return state.Draggable.Layout(gtx,
			func(gtx C) D { return content(gtx, state) },
			func(gtx C) D {
				return widget.Border{Color: th.ContrastBg, Width: unit.Dp(1), CornerRadius: unit.Dp(6)}.Layout(gtx, func(gtx C) D {
					return layout.UniformInset(unit.Dp(8)).Layout(gtx, material.Body2(th.Theme, entry.Node.Name()).Layout)
				})
			},
		)
	})
	call := macro.Stop()

	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	if state.Cutted {
		defer paint.PushOpacity(gtx.Ops, 0.6).Pop()
	}
	event.Op(gtx.Ops, entry.Node)
	pointer.CursorPointer.Add(gtx.Ops)
	call.Add(gtx.Ops)
	return dims
}

func (l *Library) layoutEntryName(gtx C, th *theme.Theme, state *filetree.NodeState) D {
	gtx.Constraints.Min.X = 0
	return state.Editable.Layout(gtx, th.Theme)
}

func (l *Library) layoutEntryIcon(gtx C, th *theme.Theme, entry filetree.FocusedEntry, size unit.Sp) D {
	color := th.ContrastBg
	if state := l.tree.GetState(entry.Node.Path); state.Marker != nil {
		color = state.Marker.Color(color)
	}
	if entry.IsNotebook {
		return l.layoutNotebook(gtx, th, size)
	}
	if entry.Node.IsDir() {
		return openFolder.Layout(gtx, color, size)
	}
	return filetree.ChooseFileIcon(entry.Node.Name()).Layout(gtx, color, size)
}

func (l *Library) layoutNotebook(gtx C, th *theme.Theme, size unit.Sp) D {
	width := gtx.Sp(size)
	height := int(float32(width) * 1.25)
	gtx.Constraints = layout.Exact(image.Pt(width, height))
	rect := image.Rectangle{Max: gtx.Constraints.Max}
	paint.FillShape(gtx.Ops, misc.WithAlpha(th.ContrastBg, 0x68), clip.UniformRRect(rect, gtx.Dp(unit.Dp(6))).Op(gtx.Ops))
	spineWidth := max(7, width/10)
	paint.FillShape(gtx.Ops, misc.WithAlpha(th.Fg, 0x38), clip.Rect{Max: image.Pt(spineWidth, height)}.Op())
	paint.FillShape(gtx.Ops, misc.WithAlpha(th.Fg, 0x48), clip.Rect{Min: image.Pt(spineWidth, 0), Max: image.Pt(spineWidth+2, height)}.Op())
	lineStart := spineWidth + width/9
	lineEnd := width - width/10
	paint.FillShape(gtx.Ops, misc.WithAlpha(th.Fg, 0x68), clip.Rect{Min: image.Pt(lineStart, height/5), Max: image.Pt(lineEnd, height/5+1)}.Op())
	paint.FillShape(gtx.Ops, misc.WithAlpha(th.Fg, 0x40), clip.Rect{Min: image.Pt(lineStart, height/5+7), Max: image.Pt(lineStart+(lineEnd-lineStart)*2/3, height/5+8)}.Op())
	return D{Size: rect.Max}
}

func (l *Library) entryKind(entry filetree.FocusedEntry) string {
	if entry.IsNotebook {
		return i18n.Translate("Notebook")
	}
	if entry.Node.IsDir() {
		return i18n.Translate("Folder")
	}
	return i18n.Translate("File")
}

func (l *Library) visibleEntries() []filetree.FocusedEntry {
	return filterLibraryEntries(l.entries, l.filter)
}

func (l *Library) filterLabel() string {
	switch l.filter {
	case libraryFilterTypst:
		return i18n.Translate("Typst work")
	case libraryFilterNotebooks:
		return i18n.Translate("Notebooks")
	case libraryFilterDocuments:
		return i18n.Translate("Typst documents")
	case libraryFilterPDFs:
		return i18n.Translate("PDFs")
	case libraryFilterImages:
		return i18n.Translate("Images")
	case libraryFilterOther:
		return i18n.Translate("Other files")
	default:
		return i18n.Translate("All files")
	}
}

func (l *Library) filterItems() []widgets.PopupWidget {
	options := []struct {
		value libraryFilter
		label string
	}{
		{libraryFilterAll, i18n.Translate("All files")},
		{libraryFilterTypst, i18n.Translate("Typst work")},
		{libraryFilterNotebooks, i18n.Translate("Notebooks")},
		{libraryFilterDocuments, i18n.Translate("Typst documents")},
		{libraryFilterPDFs, i18n.Translate("PDFs")},
		{libraryFilterImages, i18n.Translate("Images")},
		{libraryFilterOther, i18n.Translate("Other files")},
	}
	items := make([]widgets.PopupWidget, 0, len(options))
	for _, option := range options {
		option := option
		items = append(items, libraryFilterItem{
			name:     option.label,
			selected: l.filter == option.value,
			onClick:  func() { l.filter = option.value },
		})
	}
	return items
}

func filterLibraryEntries(entries []filetree.FocusedEntry, filter libraryFilter) []filetree.FocusedEntry {
	if filter == libraryFilterAll {
		return entries
	}
	filtered := make([]filetree.FocusedEntry, 0, len(entries))
	for _, entry := range entries {
		if matchesLibraryFilter(entry, filter) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func matchesLibraryFilter(entry filetree.FocusedEntry, filter libraryFilter) bool {
	if entry.Node.IsDir() {
		return !entry.IsNotebook || filter == libraryFilterTypst || filter == libraryFilterNotebooks
	}
	ext := strings.ToLower(filepath.Ext(entry.Node.Name()))
	switch filter {
	case libraryFilterTypst, libraryFilterDocuments:
		return ext == ".typ"
	case libraryFilterNotebooks:
		return false
	case libraryFilterPDFs:
		return ext == ".pdf"
	case libraryFilterImages:
		return isLibraryImage(ext)
	case libraryFilterOther:
		return ext != ".typ" && ext != ".pdf" && !isLibraryImage(ext)
	default:
		return true
	}
}

func isLibraryImage(ext string) bool {
	switch ext {
	case ".bmp", ".gif", ".ico", ".jpeg", ".jpg", ".png", ".svg", ".tif", ".tiff", ".webp":
		return true
	default:
		return false
	}
}

func (l *Library) Close() {
	if l.tree != nil {
		l.tree.Close()
	}
}

type libraryMenuItem struct {
	name    string
	onClick func()
}

type libraryFilterItem struct {
	name     string
	selected bool
	onClick  func()
}

func (i libraryFilterItem) OnClicked() {
	if i.onClick != nil {
		i.onClick()
	}
}

func (i libraryFilterItem) Layout(gtx C, th *theme.Theme) D {
	return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Right: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
					if i.selected {
						return libraryCheckIcon.Layout(gtx, th.ContrastBg, th.TextSize)
					}
					size := gtx.Sp(th.TextSize)
					return D{Size: image.Pt(size, size)}
				})
			}),
			layout.Rigid(material.Body2(th.Theme, i.name).Layout),
		)
	})
}

func (i libraryMenuItem) OnClicked() {
	if i.onClick != nil {
		i.onClick()
	}
}

func (i libraryMenuItem) Layout(gtx C, th *theme.Theme) D {
	return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(12), Right: unit.Dp(12)}.Layout(gtx, material.Body2(th.Theme, i.name).Layout)
}
