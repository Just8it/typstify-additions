package dialog

import (
	"path/filepath"
	"testing"

	"github.com/oligo/gioview/view"
)

func TestCreateProjectDialogUsesFixedLibraryLocation(t *testing.T) {
	d := &CreateProjectDialog{}
	dir := filepath.Join(t.TempDir(), "Course")

	err := d.OnInit(view.Intent{Params: map[string]any{
		ProjectDirParam:      dir,
		FixedProjectDirParam: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if d.projectDir != filepath.Clean(dir) {
		t.Fatalf("project directory = %q, want %q", d.projectDir, filepath.Clean(dir))
	}
	if !d.fixedProjectDir {
		t.Fatal("Library project location should be fixed")
	}
}

func TestCreateProjectDialogDoesNotFixMissingLocation(t *testing.T) {
	d := &CreateProjectDialog{}

	err := d.OnInit(view.Intent{Params: map[string]any{
		FixedProjectDirParam: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if d.fixedProjectDir {
		t.Fatal("missing project location must remain selectable")
	}
}
