package navpanel

import (
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"gioui.org/font"
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
	"looz.ws/typstify/service/bus"
	settingsmodel "looz.ws/typstify/service/settings"
	"looz.ws/typstify/ui/dialog"
	"looz.ws/typstify/ui/editors"
	"looz.ws/typstify/ui/statusbar"
	"looz.ws/typstify/ui/viewer"
	"looz.ws/typstify/utils"
	"looz.ws/typstify/widgets"
	"looz.ws/typstify/widgets/filetree"
	"looz.ws/typstify/widgets/icons"
	"looz.ws/typstify/widgets/menu"
)

type (
	C = layout.Context
	D = layout.Dimensions
)

type FileTreeNav struct {
	title string
	tree  *filetree.TreeView
	srv   *service.ServiceFacade
	vm    view.ViewManager

	rootSwitched bool
	newRoot      string
	pendingOpen  string
	pendingFile  string
	pendingClose string

	historyBtn      widget.Clickable
	historyProjects *RecentProjects

	contextualTree     *filetree.TreeView
	contextualFile     string
	contextualScope    string
	contextualKind     filetree.EditorScopeKind
	contextualOverride string
	focusedScope       string
	requestedScope     string
	scopeBtn           widget.Clickable
	scopePopup         *widgets.Popup
}

// Construct a FileTreeNav object that loads files and folders from rootDir. The skipFolders
// parameter allows you to specify folder name prefixes to exclude from the navigation.
func NewFileTreeNav(title string, srv *service.ServiceFacade, vm view.ViewManager) *FileTreeNav {
	ftn := &FileTreeNav{
		title:           title,
		srv:             srv,
		vm:              vm,
		historyProjects: NewRecentProjects(srv),
		scopePopup:      &widgets.Popup{MaxHeight: unit.Dp(300), Width: unit.Dp(250), Direction: layout.S},
	}

	srv.EventBus().Subscribe(ftn, "filetree", `project\.(switched|create)$`, func(topic string, data interface{}) {
		if topic == bus.TopicProjectCreate {
			created, ok := data.(bus.ProjectCreatedEvent)
			if !ok {
				log.Printf("invalid project creation event: %#v", data)
				return
			}
			if srv.Settings().FileInterface().Mode == settingsmodel.FileInterfaceModeStudent {
				ftn.pendingOpen = created.Path
				ftn.pendingClose = created.ReplacedFile
			} else if created.SwitchWorkspace {
				ftn.pendingFile = created.OpenFile
			}
			if !created.SwitchWorkspace || ftn.tree != nil && created.Path == ftn.tree.Root() {
				return
			}
			ftn.saveLastWorkplace()
			ftn.newRoot = created.Path
			return
		}

		path, ok := data.(string)
		if !ok {
			log.Printf("invalid workspace switch event: %#v", data)
			return
		}
		if ftn.tree == nil || path != ftn.tree.Root() {
			ftn.saveLastWorkplace()
			ftn.newRoot = path
		}
	})

	return ftn
}

func (tn *FileTreeNav) switchRoot() {
	if tn.newRoot == "" {
		return
	}

	newRoot, err := filepath.Abs(tn.newRoot)
	if err != nil {
		log.Println("convert dir to abs dir error: ", err)
		return
	}

	tn.focusedScope = ""
	tn.closeContextualTree()
	tn.srv.SetProjectDir(newRoot)

	// Restore the workplace.
	states := tn.srv.Workspace().Current().TreeState
	var newTree *filetree.TreeView
	if states != nil {
		restoredTree, err := filetree.RestoreTree(states)
		if err != nil {
			log.Println("Restore file tree error: ", err)
		} else {
			newTree = restoredTree
		}
	}

	if newTree == nil {
		root, err := explorer.NewFileTree(newRoot)
		if err != nil {
			log.Println("open explorer failed: ", err)
			return
		}

		newTree = filetree.NewTreeView(root)
	}

	configureFileTree(newTree, tn.srv, tn.vm)
	newTree.OnFileSelectedFunc = tn.onWorkspaceFileSelected
	newTree.OnFolderSelectedFunc = tn.onWorkspaceFolderSelected

	tn.tree = newTree

	for _, file := range tn.srv.Workspace().Current().OpenedFiles {
		node, err := explorer.NewFileTree(file)
		if err != nil {
			log.Println("open file failed: ", err)
			continue
		}
		tn.onFileSelected(node)
	}

}

func configureFileTree(tree *filetree.TreeView, srv *service.ServiceFacade, vm view.ViewManager) {
	controller := &FileTreeNav{tree: tree, srv: srv, vm: vm}
	tree.OnFileSelectedFunc = controller.onFileSelected
	tree.OnDropConfirmFunc = onDropConfirmFunc(vm, tree.Root())
	tree.OnFileUpdatedFunc = controller.onFileUpdated
	tree.OnFileRemoveFunc = controller.onFileDeleted
	tree.OnErrorFunc = func(err error) {
		log.Println("file tree error: ", err)
		srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: err.Error(), Level: 1})
	}
	tree.ExtraMenuOptionProvider = controller.extraMenuOptions
	tree.NodeMarkerProvider = controller.nodeMarker
}

func (tn *FileTreeNav) saveLastWorkplace() {
	if tn.tree == nil {
		return
	}

	defer tn.vm.Reset()

	states := tn.tree.Snapshot()
	openedFiles := make([]string, 0)
	views := tn.vm.OpenedViews()
	for _, vw := range views {
		location := vw.Location()
		switch vw.ID() {
		case editors.GenericTextEditorViewID, editors.TypstEditorViewID, viewer.ImgViewerViewID:
			filePath := location.Query().Get("path")
			if filePath != "" {
				openedFiles = append(openedFiles, filePath)
			}
		}
	}

	tn.srv.Workspace().SaveSnapshot(states, openedFiles)
}

func (tn *FileTreeNav) OnClose() {
	tn.saveLastWorkplace()
	tn.closeContextualTree()
	if tn.tree != nil {
		tn.tree.Close()
	}
}

func (tn *FileTreeNav) Title() string {
	return tn.title
}

func (tn *FileTreeNav) Icon() *icons.SvgIcon {
	return explorerIcon
}

func (tn *FileTreeNav) FocusScope(scope string) {
	tn.focusedScope = filepath.Clean(scope)
}

func (tn *FileTreeNav) LayoutHeader(gtx C, th *theme.Theme) D {
	tn.Update(gtx)
	if tn.contextualTree != nil {
		label := material.Caption(th.Theme, i18n.Translate("Explorer"))
		label.Font.Weight = font.Medium
		return label.Layout(gtx)
	}

	if tn.historyBtn.Clicked(gtx) {
		tn.historyProjects.Show()
	}

	title := i18n.Translate("No project")
	if tn.tree != nil {
		root := filepath.Base(tn.tree.Root())
		if root != "" {
			title = root
		}
	}

	title = strings.ToUpper(title)

	return tn.historyProjects.Layout(gtx, th,
		func(gtx C) D {
			return tn.historyBtn.Layout(gtx, func(gtx C) D {
				macro := op.Record(gtx.Ops)
				dims := layout.Inset{
					Top:    unit.Dp(2),
					Bottom: unit.Dp(2),
					Left:   unit.Dp(4),
					Right:  unit.Dp(4),
				}.Layout(gtx, func(gtx C) D {
					return material.Subtitle2(th.Theme, title).Layout(gtx)
				})
				callOp := macro.Stop()

				defer clip.UniformRRect(
					image.Rectangle{
						Max: dims.Size,
					},
					gtx.Dp(unit.Dp(4)),
				).Push(gtx.Ops).Pop()

				if tn.historyBtn.Hovered() {
					pointer.CursorPointer.Add(gtx.Ops)
					paint.ColorOp{Color: misc.WithAlpha(th.ContrastBg, th.HoverAlpha)}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)

				}

				callOp.Add(gtx.Ops)

				return dims
			})
		},
	)

}

func (tn *FileTreeNav) Update(gtx C) bool {
	updated := tn.newRoot != ""
	if tn.newRoot != "" {
		tn.switchRoot()
	}

	tn.newRoot = ""
	if tn.pendingClose != "" {
		tn.closeOpenedFile(tn.pendingClose)
		tn.pendingClose = ""
	}
	if tn.pendingOpen != "" {
		path := tn.pendingOpen
		tn.pendingOpen = ""
		tn.openProject(path)
	}
	if tn.pendingFile != "" {
		path := tn.pendingFile
		tn.pendingFile = ""
		tn.openFile(path)
	}
	tn.updateContextualTree()
	return updated
}

func (tn *FileTreeNav) openFile(path string) {
	node, err := explorer.NewFileTree(path)
	if err != nil {
		log.Printf("opening file %s: %v", path, err)
		return
	}
	tn.onFileSelected(node)
}

func (tn *FileTreeNav) closeOpenedFile(path string) {
	views := tn.vm.OpenedViews()
	for idx := len(views) - 1; idx >= 0; idx-- {
		location := views[idx].Location()
		if location.Query().Get("path") == path {
			tn.vm.CloseTab(idx)
		}
	}
}

func (tn *FileTreeNav) Layout(gtx C, th *theme.Theme) D {
	tn.Update(gtx)

	if tn.contextualTree != nil {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return tn.layoutContextualScope(gtx, th) }),
			layout.Flexed(1, func(gtx C) D { return tn.contextualTree.Layout(gtx, th) }),
		)
	}
	tree := tn.tree
	if tree == nil {
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx C) D {
			lb := material.Label(th.Theme, th.TextSize*0.9, i18n.Translate("No open project."))
			lb.Font.Style = font.Italic
			lb.Color = misc.WithAlpha(th.Fg, 0xb6)
			return lb.Layout(gtx)
		})
	}

	return tree.Layout(gtx, th)
}

func (tn *FileTreeNav) updateContextualTree() {
	setting := tn.srv.Settings().FileInterface()
	if setting.Mode != settingsmodel.FileInterfaceModeStudent {
		tn.focusedScope = ""
		tn.closeContextualTree()
		return
	}
	if tn.tree == nil {
		tn.closeContextualTree()
		return
	}

	current := tn.vm.CurrentView()
	if current == nil {
		tn.closeContextualTree()
		return
	}
	location := current.Location()
	file := location.Query().Get("path")
	if file == "" {
		tn.closeContextualTree()
		return
	}
	focused := tn.focusedScope != "" && filetree.PathWithin(tn.focusedScope, file)
	if tn.focusedScope != "" && !focused {
		tn.focusedScope = ""
	}
	if setting.EditorScope != settingsmodel.FileInterfaceEditorScopeContextual && !focused {
		tn.closeContextualTree()
		return
	}

	fileChanged := filepath.Clean(file) != filepath.Clean(tn.contextualFile)
	if fileChanged {
		tn.contextualOverride = ""
	}
	scope, kind, err := filetree.ResolveEditorScope(tn.tree.Root(), file)
	if err != nil {
		tn.closeContextualTree()
		return
	}
	if tn.requestedScope != "" {
		tn.contextualOverride = tn.requestedScope
		tn.requestedScope = ""
	}
	if focused {
		scope = tn.focusedScope
		kind = scopeKind(scope)
	}
	if tn.contextualOverride != "" {
		scope = tn.contextualOverride
		kind = scopeKind(scope)
	}

	tn.contextualFile = file
	if tn.contextualTree != nil && filepath.Clean(scope) == filepath.Clean(tn.contextualScope) {
		tn.contextualKind = kind
		if fileChanged {
			_ = tn.contextualTree.SelectPath(file)
		}
		return
	}

	root, err := explorer.NewFileTree(scope)
	if err != nil {
		tn.closeContextualTree()
		return
	}
	tree := filetree.NewTreeView(root)
	tree.HideRoot = true
	configureFileTree(tree, tn.srv, tn.vm)
	_ = tree.SelectPath(file)
	if tn.contextualTree != nil {
		tn.contextualTree.Close()
	}
	tn.contextualTree = tree
	tn.contextualFile = file
	tn.contextualScope = scope
	tn.contextualKind = kind
}

func (tn *FileTreeNav) closeContextualTree() {
	if tn.contextualTree != nil {
		tn.contextualTree.Close()
	}
	tn.contextualTree = nil
	tn.contextualFile = ""
	tn.contextualScope = ""
	tn.contextualKind = ""
	tn.contextualOverride = ""
	tn.requestedScope = ""
}

func scopeKind(path string) filetree.EditorScopeKind {
	marker, err := os.Stat(filepath.Join(path, "typst.toml"))
	if err == nil && !marker.IsDir() {
		return filetree.EditorScopeProject
	}
	return filetree.EditorScopeFolder
}

func (tn *FileTreeNav) layoutContextualScope(gtx C, th *theme.Theme) D {
	if tn.scopeBtn.Clicked(gtx) {
		tn.scopePopup.SetOpen()
	}
	items := make([]widgets.PopupWidget, 0)
	workspaceRoot := filepath.Clean(tn.tree.Root())
	for path := filepath.Clean(tn.contextualScope); ; path = filepath.Dir(path) {
		path := path
		items = append(items, scopeMenuItem{
			name: filepath.Base(path),
			kind: scopeKind(path),
			onClick: func() {
				tn.requestedScope = path
			},
		})
		if path == workspaceRoot {
			break
		}
	}

	return tn.scopePopup.Layout(gtx, th, func(gtx C) D {
		return layout.Inset{Top: unit.Dp(9), Bottom: unit.Dp(10), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
			return tn.scopeBtn.Layout(gtx, func(gtx C) D {
				return widget.Border{Color: misc.WithAlpha(th.Fg, 0x24), Width: unit.Dp(1), CornerRadius: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
					return layout.Inset{Top: unit.Dp(7), Bottom: unit.Dp(7), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx C) D {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
							layout.Rigid(func(gtx C) D {
								return explorerIcon.Layout(gtx, th.ContrastBg, th.TextSize)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Flexed(1, func(gtx C) D {
								label := material.Subtitle2(th.Theme, filepath.Base(tn.contextualScope))
								label.MaxLines = 1
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx C) D {
								label := material.Caption(th.Theme, strings.ToUpper(string(tn.contextualKind)))
								label.Color = th.ContrastBg
								return layout.Inset{Left: unit.Dp(5), Right: unit.Dp(5)}.Layout(gtx, label.Layout)
							}),
							layout.Rigid(func(gtx C) D {
								return arrowDownIcon.Layout(gtx, th.Fg, th.TextSize)
							}),
						)
					})
				})
			})
		})
	}, items...)
}

type scopeMenuItem struct {
	name    string
	kind    filetree.EditorScopeKind
	onClick func()
}

func (s scopeMenuItem) OnClicked() { s.onClick() }

func (s scopeMenuItem) Layout(gtx C, th *theme.Theme) D {
	return layout.Inset{Top: unit.Dp(5), Bottom: unit.Dp(5), Left: unit.Dp(10), Right: unit.Dp(10)}.Layout(gtx, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Flexed(1, material.Body2(th.Theme, s.name).Layout),
			layout.Rigid(func(gtx C) D {
				label := material.Caption(th.Theme, strings.ToUpper(string(s.kind)))
				label.Color = misc.WithAlpha(th.Fg, 0x90)
				return label.Layout(gtx)
			}),
		)
	})
}

// onFileUpdated close opened view, and then re-open the updated file.
func (tn *FileTreeNav) onFileUpdated(node *filetree.FileNode, oldPath string) {
	if node.IsDir() {
		return
	}

	views := tn.vm.OpenedViews()
	for idx, vw := range views {
		location := vw.Location()
		switch vw.ID() {
		case editors.GenericTextEditorViewID, editors.TypstEditorViewID, viewer.ImgViewerViewID:
			filePath := location.Query().Get("path")
			if filePath == oldPath {
				tn.vm.CloseTab(idx)
			}
		}
	}

	tn.vm.RequestSwitch(onFileSelected(node))
}

func (tn *FileTreeNav) onFileSelected(node *filetree.FileNode) {
	if node == nil {
		return
	}

	exists, isDir := utils.CheckFileExists(node.Path)
	if !exists || isDir {
		return
	}

	intent := onFileSelected(node)
	// An empty also refresh the UI so do not drop it.
	if err := tn.vm.RequestSwitch(intent); err != nil {
		log.Printf("switching to view %s error: %v", intent.Target, err)
		tn.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: err.Error(), Level: 1})
	}
}

func (tn *FileTreeNav) onWorkspaceFileSelected(node *filetree.FileNode) {
	if node != nil && tn.srv.Settings().FileInterface().Mode == settingsmodel.FileInterfaceModeStudent {
		tn.FocusScope(filepath.Dir(node.Path))
	}
	tn.onFileSelected(node)
}

func (tn *FileTreeNav) onWorkspaceFolderSelected(node *filetree.FileNode) {
	if node == nil || tn.srv.Settings().FileInterface().Mode != settingsmodel.FileInterfaceModeStudent || !isPackageProject(node.Path) {
		return
	}
	tn.openProject(node.Path)
}

func (tn *FileTreeNav) openProject(path string) {
	entrypoint, err := filetree.NotebookEntrypoint(path)
	if err != nil {
		log.Printf("opening notebook %s: %v", path, err)
		tn.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: err.Error(), Level: 1})
		return
	}
	entry, err := explorer.NewFileTree(entrypoint)
	if err != nil {
		log.Printf("opening notebook entrypoint %s: %v", entrypoint, err)
		return
	}
	tn.FocusScope(path)
	tn.onFileSelected(entry)
}

func (tn *FileTreeNav) onFileDeleted(node *filetree.FileNode) {
	rootDir := tn.tree.Root()

	go func() {
		destPath := filepath.Clean(node.Path)
		relPath, err := filepath.Rel(rootDir, destPath)
		if err == nil {
			destPath = relPath
		}

		caller := dialog.NewDialogChooser[bool](tn.vm)
		// It's a blocking call, should call it on a separated goroutine.
		result, err := caller.Call(dialog.DeleteFileDialogViewID, map[string]any{"destination": destPath})
		if err != nil {
			log.Println("delete file error: ", err)
		}

		if result.Params {
			tn.tree.Remove(node)
		}
	}()
}

func (tn *FileTreeNav) nodeMarker(nodePath string) *filetree.NodeMarker {
	// First check if it's managed bibliography file.
	settings := tn.srv.Workspace().LoadWorkspaceSettings()
	if len(settings.BibFiles) > 0 {
		root := tn.srv.Workspace().Current().Path
		if root == "" {
			root = tn.tree.Root()
		}
		relPath, err := filepath.Rel(root, nodePath)
		if err != nil {
			log.Println("get relative path error: ", err)
			return nil
		}

		idx := slices.IndexFunc(settings.BibFiles, func(bib service.ManagedBibliography) bool {
			return bib.File == relPath
		})
		if idx >= 0 {
			return &filetree.NodeMarker{
				Kind:  "bib",
				Color: func(baseColor color.NRGBA) color.NRGBA { return utils.DisableColor(baseColor) },
				Meta: map[string]any{
					"meta": settings.BibFiles[idx],
				},
			}
		}
	}

	// Then check if its git managed and has changes made.

	// return &filetree.NodeMarker{
	// 	Kind:  "git",
	// 	Color: func(th *theme.Theme) color.NRGBA { return misc.WithAlpha(th.ContrastBg, 0x60) },
	// }

	// It's just regular node, do not set a marker.
	return nil
}

func (tn *FileTreeNav) extraMenuOptions(node *filetree.FileNode) [][]menu.MenuOption {
	isPackage := isPackageProject(tn.tree.Root())
	isTpixLoggedIn := tn.srv.TpixSessionService().Authenticated()
	publishPackageOpt := menu.MenuOption{
		OnClicked: func(gtx layout.Context) error {
			if !isPackage || !isTpixLoggedIn {
				return nil
			}

			// open the publish dialog
			tn.vm.RequestSwitch(view.Intent{
				Target:      dialog.PublishPkgDialogViewID,
				ShowAsModal: true,
				Params:      map[string]any{"projectDir": tn.tree.Root()},
			})

			return nil
		},

		Layout: func(gtx layout.Context, th *theme.Theme) layout.Dimensions {
			name := i18n.Translate("Publish Package")
			label := material.Label(th.Theme, th.TextSize, name)
			if !isPackage || !isTpixLoggedIn {
				label.Color = utils.DisableColor(th.Fg)
			}
			return label.Layout(gtx)
		},
	}

	syncDependenciesOpt := menu.MenuOption{
		OnClicked: func(gtx layout.Context) error {
			if !isTpixLoggedIn {
				return nil
			}

			go func() {
				err := tn.srv.PkgService().PullDependencies(tn.tree.Root())
				if err != nil {
					tn.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: i18n.Translate("pull dependencies error: %s", err.Error()), Level: 2})
					return
				}
				tn.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{Content: i18n.Translate("pull dependencies succeeded!")})

			}()
			return nil
		},

		Layout: func(gtx layout.Context, th *theme.Theme) layout.Dimensions {
			name := i18n.Translate("Sync Dependencies")
			label := material.Label(th.Theme, th.TextSize, name)
			if !isTpixLoggedIn {
				label.Color = utils.DisableColor(th.Fg)
			}
			return label.Layout(gtx)
		},
	}

	syncBibOpt := menu.MenuOption{
		OnClicked: func(gtx layout.Context) error {
			if !isTpixLoggedIn {
				return nil
			}

			root := tn.srv.Workspace().Current().Path
			if root == "" {
				root = tn.tree.Root()
			}
			relPath, _ := filepath.Rel(root, node.Path)
			// open the publish dialog
			tn.vm.RequestSwitch(view.Intent{
				Target:      dialog.SyncBibDialogViewID,
				ShowAsModal: true,
				Params:      map[string]any{"parentDir": relPath},
			})

			return nil
		},

		Layout: func(gtx layout.Context, th *theme.Theme) layout.Dimensions {
			name := i18n.Translate("Sync Bibliographies")
			label := material.Label(th.Theme, th.TextSize, name)
			if !isTpixLoggedIn {
				label.Color = utils.DisableColor(th.Fg)
			}
			return label.Layout(gtx)
		},
	}

	bibInfoOpt := menu.MenuOption{
		OnClicked: func(gtx layout.Context) error {
			if !isTpixLoggedIn {
				return nil
			}

			state := tn.tree.GetState(node.Path)
			if state.Marker == nil || state.Marker.Kind != "bib" {
				return nil
			}

			meta := state.Marker.Meta["meta"]

			// open the info dialog
			tn.vm.RequestSwitch(view.Intent{
				Target:      dialog.ViewBibInfoDialogViewID,
				ShowAsModal: true,
				Params:      map[string]any{"meta": meta},
			})

			return nil
		},

		Layout: func(gtx layout.Context, th *theme.Theme) layout.Dimensions {
			name := i18n.Translate("View Bibliography Info")
			label := material.Label(th.Theme, th.TextSize, name)
			if !isTpixLoggedIn {
				label.Color = utils.DisableColor(th.Fg)
			}
			return label.Layout(gtx)
		},
	}

	options := [][]menu.MenuOption{}
	if tn.tree.Root() == node.Path {
		options = append(options, []menu.MenuOption{publishPackageOpt, syncDependenciesOpt})
	}

	if node.IsDir() {
		options = append(options, []menu.MenuOption{syncBibOpt})
	} else {
		state := tn.tree.GetState(node.Path)
		if state.Marker != nil && state.Marker.Kind == "bib" {
			options = append(options, []menu.MenuOption{bibInfoOpt})
		}
	}

	return options
}

func isPackageProject(projectDir string) bool {
	manifestPath := filepath.Join(projectDir, "typst.toml")
	if _, err := os.Stat(manifestPath); err != nil {
		return false
	}

	return true

}

func onDropConfirmFunc(vm view.ViewManager, rootDir string) filetree.OnDropConfirmFunc {
	return func(srcPath string, dest *filetree.FileNode, onConfirm func()) {
		go func() {
			caller := dialog.NewDialogChooser[bool](vm)
			srcPath = filepath.Clean(srcPath)
			relPath, err := filepath.Rel(rootDir, srcPath)
			if err != nil {
				log.Printf("Error calculating relative path: %v\n", err)
			} else {
				srcPath = relPath
			}

			result, err := caller.Call(dialog.DndDropFileDialogViewID, map[string]any{"source": srcPath, "destination": dest.Name()})
			if err != nil {
				log.Println("DnD dialog error: ", err)
				return
			}

			if result.Params {
				onConfirm()
			}
		}()
	}
}

func onFileSelected(node *filetree.FileNode) view.Intent {
	if slices.Contains([]string{".png", ".jpg", ".jpeg", ".gif", ".PNG", ".JPG", ".JPEG", ".GIF"}, node.FileType()) {
		return view.Intent{
			Target:      viewer.ImgViewerViewID,
			ShowAsModal: false,
			RequireNew:  true,
			Params: map[string]interface{}{
				"path": node.Path,
			},
		}
	}

	if node.FileType() == ".typ" {
		return view.Intent{
			Target:      editors.TypstEditorViewID,
			ShowAsModal: false,
			RequireNew:  true,
			Params: map[string]interface{}{
				"path": node.Path,
			},
		}
	}

	// detect its MIME type to see if it's a text file.
	if utils.IsTextFile(node.Path) {
		// open as plain text
		return view.Intent{
			Target:      editors.GenericTextEditorViewID,
			ShowAsModal: false,
			RequireNew:  true,
			Params: map[string]interface{}{
				"path": node.Path,
			},
		}
	}

	return view.Intent{
		Target:      dialog.OpenWithExternalAppDialogViewID,
		ShowAsModal: true,
		Params: map[string]interface{}{
			"path": node.Path,
		},
	}

}
