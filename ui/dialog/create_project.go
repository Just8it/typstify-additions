package dialog

import (
	"errors"
	"log"
	"path/filepath"
	"strings"
	"time"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gioview/view"
	gw "github.com/oligo/gioview/widget"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/service"
	"looz.ws/typstify/service/bus"
	"looz.ws/typstify/typst"
	"looz.ws/typstify/ui/statusbar"
	"looz.ws/typstify/widgets/icons"
)

type ProjectKind string

const (
	DocumentKind ProjectKind = "Document"
	PackageKind  ProjectKind = "Package"
	TemplateKind ProjectKind = "Template"
)

type ProjectCreateReq struct {
	Kind       ProjectKind
	ProjectDir string
	Name       string
	// template to use, for package or template kind this should be omitted.
	TemplateName string
}

type CreateProjectDialog struct {
	srv             *service.ServiceFacade
	kindEnum        widget.Enum
	templateInput   gw.TextField
	nameInput       gw.TextField
	projectDir      string
	fixedProjectDir bool
	projectDirInput gw.TextField
	openFolderBtn   widget.Clickable

	kindChoices []layout.FlexChild
}

var CreateProjectDialogViewID = view.NewViewID("CreateProjectDialogView")

const (
	ProjectDirParam      = "projectDir"
	FixedProjectDirParam = "fixedProjectDir"
)

var projectKinds = []ProjectKind{DocumentKind, PackageKind, TemplateKind}

var (
	folderOpenIcon = icons.NewSvgIcon(icons.FolderOpen)
)

func NewCreateProjectDialog(srv *service.ServiceFacade) view.View {
	createDialog := &CreateProjectDialog{srv: srv}

	dialog := NewDialogModal(CreateProjectDialogViewID, i18n.Translate("Create New Project"), i18n.Translate("Create"))
	dialog.Dialog = createDialog
	return dialog
}

func (d *CreateProjectDialog) OnInit(intent view.Intent) error {
	d.kindEnum.Value = string(DocumentKind)
	d.projectDir = ""
	d.fixedProjectDir = false

	if projectDir, ok := intent.Params[ProjectDirParam].(string); ok && projectDir != "" {
		d.projectDir = filepath.Clean(projectDir)
	}
	if fixed, ok := intent.Params[FixedProjectDirParam].(bool); ok {
		d.fixedProjectDir = fixed && d.projectDir != ""
	}
	if template, ok := intent.Params["template"].(string); ok {
		d.templateInput.SetText(template)
	}

	return nil
}

func (d *CreateProjectDialog) createDocumentProject(req *ProjectCreateReq) (string, error) {
	if req.Kind != DocumentKind {
		return "", nil
	}

	if req.Name == "" {
		return "", errors.New(i18n.Translate("Please set a name for your project."))
	}

	if req.ProjectDir == "" {
		return "", errors.New(i18n.Translate("Please open a folder in File Explorer to place the project files."))
	}

	if req.TemplateName != "" && !strings.HasPrefix(req.TemplateName, "@") {
		return "", errors.New(i18n.Translate("Please input a valid full template name."))
	}

	if req.TemplateName != "" {
		dir := filepath.Join(req.ProjectDir, req.Name)
		_, _, err := d.srv.PkgService().DownloadWithSpec(req.TemplateName)
		if err != nil {
			return "", err
		}
		err = typst.InitCmd(req.TemplateName, dir, &typst.InitCmdOptions{
			PackagePath:      d.srv.Settings().Typst().PackageDir,
			PackageCachePath: d.srv.Settings().Typst().PackageCacheDir,
		})
		if err != nil {
			return "", err
		}
		return dir, nil

	} else {
		return d.srv.PkgService().CreateSampleDocument(req.ProjectDir, req.Name)
	}

}

func (d *CreateProjectDialog) createPackageProject(req *ProjectCreateReq) (string, error) {
	if req.Kind != PackageKind && req.Kind != TemplateKind {
		return "", errors.New(i18n.Translate("Not a package creation request."))
	}

	if req.Name == "" {
		return "", errors.New(i18n.Translate("Please set a name for your package/template."))
	}

	dir, err := d.srv.PkgService().CreatePkg(req.ProjectDir, req.Name, req.Kind == TemplateKind)
	if err != nil {
		log.Println("create package error: ", err)
	}

	return dir, nil
}

func (d *CreateProjectDialog) OnConfirm() error {
	name := strings.TrimSpace(d.nameInput.Text())
	if name == "" {
		return errors.New(i18n.Translate("Please set a name for your project."))
	}
	if d.projectDir == "" {
		return errors.New(i18n.Translate("Please select where the project should be created."))
	}

	req := ProjectCreateReq{
		Kind:       ProjectKind(d.kindEnum.Value),
		ProjectDir: d.projectDir,
		Name:       name,
	}
	if req.Kind == DocumentKind {
		req.TemplateName = strings.TrimSpace(d.templateInput.Text())
		if req.TemplateName != "" && !strings.HasPrefix(req.TemplateName, "@") {
			return errors.New(i18n.Translate("Please input a valid full template name."))
		}
	}

	go func(req ProjectCreateReq) {
		var err error
		var dir string

		switch req.Kind {
		case DocumentKind:
			dir, err = d.createDocumentProject(&req)
		case PackageKind:
			dir, err = d.createPackageProject(&req)
		case TemplateKind:
			dir, err = d.createPackageProject(&req)
		}

		if err != nil {
			d.srv.EventBus().Emit(bus.TopicStatusbarNotifyEvent, statusbar.Notification{
				Content:  i18n.Translate("Create project error: %s", err.Error()),
				Level:    2,
				Duration: time.Second * 8,
			})
			return
		}

		d.srv.EventBus().Emit(bus.TopicProjectCreate, bus.ProjectCreatedEvent{
			Path:            dir,
			SwitchWorkspace: !d.fixedProjectDir,
		})
	}(req)

	return nil
}

func (d *CreateProjectDialog) LayoutBody(gtx C, th *theme.Theme) D {
	if !d.fixedProjectDir && d.openFolderBtn.Clicked(gtx) {
		go func() {
			d.projectDir, _ = d.srv.FileChooser().ChooseFolder()
		}()
	}

	if d.kindChoices == nil {
		for _, name := range projectKinds {
			name := name
			d.kindChoices = append(d.kindChoices, layout.Rigid(func(gtx C) D {
				return material.RadioButton(th.Theme, &d.kindEnum, string(name), projectKindLabel(name)).Layout(gtx)
			}))
		}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return formItem{Axis: layout.Vertical}.Layout(gtx, th, i18n.Translate("Project Type"), i18n.Translate("Choose Document for notes, articles, books, or slides. Choose Package or Template only when developing reusable Typst packages."),
				func(gtx C) D {
					return layout.Flex{
						Axis: layout.Horizontal,
					}.Layout(gtx, d.kindChoices...)
				})
		}),

		layout.Rigid(func(gtx C) D {
			title := i18n.Translate("Project Location")
			description := i18n.Translate("Select the folder where the project will be created.")
			if d.fixedProjectDir {
				title = i18n.Translate("Create In")
				description = i18n.Translate("The project will be created inside the current Library folder.")
			}

			return formItem{Axis: layout.Vertical}.Layout(gtx, th, title, description,
				func(gtx C) D {
					locationField := func(gtx C) D {
						d.projectDirInput.Alignment = text.Start
						d.projectDirInput.SingleLine = true
						d.projectDirInput.State().ReadOnly = true
						d.projectDirInput.Leading = func(gtx layout.Context) layout.Dimensions {
							return folderOpenIcon.Layout(gtx, th.ContrastBg, th.TextSize*1.2)
						}
						if d.projectDir != d.projectDirInput.Text() {
							d.projectDirInput.SetText(d.projectDir)
						}

						return d.projectDirInput.Layout(gtx, th, i18n.Translate("Select a directory"))
					}
					if d.fixedProjectDir {
						return locationField(gtx)
					}
					return d.openFolderBtn.Layout(gtx, locationField)
				})
		}),

		layout.Rigid(func(gtx C) D {
			if d.kindEnum.Value != string(DocumentKind) {
				return D{}
			}
			return formItem{Axis: layout.Vertical}.Layout(gtx, th, i18n.Translate("Typst Template"), i18n.Translate("Optionally start from a Typst package such as @preview/aero-check:0.1.1. Leave this empty to create a basic document."), func(gtx C) D {
				d.templateInput.Alignment = text.Start
				return d.templateInput.Layout(gtx, th, i18n.Translate("@preview/package:version (optional)"))
			})
		}),

		layout.Rigid(func(gtx C) D {
			return formItem{Axis: layout.Vertical}.Layout(gtx, th, i18n.Translate("Project Name"),
				i18n.Translate("A folder with this name will be created at the location above."),
				func(gtx C) D {
					d.nameInput.Alignment = text.Start
					d.nameInput.SingleLine = true
					return d.nameInput.Layout(gtx, th, i18n.Translate("Project name"))
				})
		}),
	)

}

func projectKindLabel(kind ProjectKind) string {
	switch kind {
	case PackageKind:
		return i18n.Translate("Package")
	case TemplateKind:
		return i18n.Translate("Template")
	default:
		return i18n.Translate("Document")
	}
}
