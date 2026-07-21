package navpanel

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"looz.ws/typstify/widgets/filetree"
)

func TestFilterLibraryEntries(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"Folder", "Notebook"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"notes.typ", "handout.pdf", "diagram.PNG", "readme.txt", filepath.Join("Notebook", "typst.toml")} {
		if err := os.WriteFile(filepath.Join(root, file), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	browser, err := filetree.NewFocusedBrowser(root, "")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := browser.Entries()
	if err != nil {
		t.Fatal(err)
	}

	tests := map[libraryFilter][]string{
		libraryFilterTypst:     {"Folder", "Notebook", "notes.typ"},
		libraryFilterNotebooks: {"Folder", "Notebook"},
		libraryFilterDocuments: {"Folder", "notes.typ"},
		libraryFilterPDFs:      {"Folder", "handout.pdf"},
		libraryFilterImages:    {"Folder", "diagram.PNG"},
		libraryFilterOther:     {"Folder", "readme.txt"},
	}
	for filter, want := range tests {
		gotEntries := filterLibraryEntries(entries, filter)
		got := make([]string, 0, len(gotEntries))
		for _, entry := range gotEntries {
			got = append(got, entry.Node.Name())
		}
		slices.Sort(got)
		slices.Sort(want)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("filter %q: got %v, want %v", filter, got, want)
		}
	}
}
