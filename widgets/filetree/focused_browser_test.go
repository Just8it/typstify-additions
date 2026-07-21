package filetree

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oligo/gioview/explorer"
)

func TestFocusedBrowser(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "Course")
	grandchild := filepath.Join(child, "Lecture")
	if err := os.MkdirAll(grandchild, 0755); err != nil {
		t.Fatal(err)
	}

	browser, err := NewFocusedBrowser(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := browser.Enter(child); err != nil {
		t.Fatal(err)
	}
	if err := browser.Enter(grandchild); err != nil {
		t.Fatal(err)
	}
	crumbs := browser.Breadcrumbs()
	if len(crumbs) != 3 || crumbs[1].Path != child || crumbs[2].Path != grandchild {
		t.Fatalf("breadcrumbs = %#v", crumbs)
	}
	if err := browser.Up(); err != nil || browser.Current() != child {
		t.Fatalf("up: current = %q, err = %v", browser.Current(), err)
	}
	if err := browser.Enter(crumbs[0].Path); err != nil || browser.Current() != root {
		t.Fatalf("breadcrumb: current = %q, err = %v", browser.Current(), err)
	}

	file := filepath.Join(root, "notes.typ")
	if err := os.WriteFile(file, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := browser.Enter(file); err == nil {
		t.Fatal("entering a file succeeded")
	}
	outside := t.TempDir()
	if err := browser.Enter(outside); err == nil {
		t.Fatal("entering outside the workspace succeeded")
	}

	otherRoot := t.TempDir()
	otherChild := filepath.Join(otherRoot, "A", "B")
	if err := os.MkdirAll(otherChild, 0755); err != nil {
		t.Fatal(err)
	}
	if err := browser.SetWorkspace(otherRoot, grandchild); err != nil {
		t.Fatal(err)
	}
	if browser.Current() != otherRoot {
		t.Fatalf("workspace current = %q, want %q", browser.Current(), otherRoot)
	}
	if err := browser.Enter(otherChild); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(otherChild); err != nil {
		t.Fatal(err)
	}
	if _, err := browser.Entries(); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Dir(otherChild); browser.Current() != want {
		t.Fatalf("deleted fallback = %q, want %q", browser.Current(), want)
	}
}

func TestFocusedBrowserEntries(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".claude", "zFolder", "aFolder", "Notebook", filepath.Join("Folder", "NestedNotebook")} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"Zoo.typ", "alpha.typ", filepath.Join("Notebook", "typst.toml"), filepath.Join("Folder", "NestedNotebook", "typst.toml")} {
		if err := os.WriteFile(filepath.Join(root, file), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	browser, err := NewFocusedBrowser(root, "")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := browser.Entries()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"aFolder", "Folder", "Notebook", "zFolder", "alpha.typ", "Zoo.typ"}
	if len(entries) != len(want) {
		t.Fatalf("entry count = %d, want %d", len(entries), len(want))
	}
	for i, name := range want {
		if entries[i].Node.Name() != name {
			t.Fatalf("entry %d = %q, want %q", i, entries[i].Node.Name(), name)
		}
	}
	if entries[2].Node.Name() != "Notebook" || !entries[2].IsNotebook {
		t.Fatalf("notebook classification = %#v", entries[2])
	}
	if entries[1].Node.Name() != "Folder" || entries[1].IsNotebook {
		t.Fatalf("nested marker classified parent as notebook: %#v", entries[1])
	}
}

func TestResolveEditorScope(t *testing.T) {
	root := t.TempDir()
	outer := filepath.Join(root, "Course")
	inner := filepath.Join(outer, "Chapter")
	loose := filepath.Join(root, "Loose")
	if err := os.MkdirAll(inner, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(loose, 0755); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{filepath.Join(outer, "typst.toml"), filepath.Join(inner, "typst.toml")} {
		if err := os.WriteFile(marker, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	projectFile := filepath.Join(inner, "notes.typ")
	looseFile := filepath.Join(loose, "notes.typ")
	for _, file := range []string{projectFile, looseFile} {
		if err := os.WriteFile(file, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}

	scope, kind, err := ResolveEditorScope(root, projectFile)
	if err != nil || scope != inner || kind != EditorScopeProject {
		t.Fatalf("nested project scope = %q, %q, %v", scope, kind, err)
	}
	scope, kind, err = ResolveEditorScope(root, looseFile)
	if err != nil || scope != loose || kind != EditorScopeFolder {
		t.Fatalf("loose file scope = %q, %q, %v", scope, kind, err)
	}
	out := filepath.Join(t.TempDir(), "outside.typ")
	if err := os.WriteFile(out, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveEditorScope(root, out); err == nil {
		t.Fatal("outside file resolved inside workspace")
	}

	boundary := t.TempDir()
	boundedRoot := filepath.Join(boundary, "Workspace")
	boundedFolder := filepath.Join(boundedRoot, "Notes")
	if err := os.MkdirAll(boundedFolder, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(boundary, "typst.toml"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	boundedFile := filepath.Join(boundedFolder, "notes.typ")
	if err := os.WriteFile(boundedFile, nil, 0644); err != nil {
		t.Fatal(err)
	}
	scope, kind, err = ResolveEditorScope(boundedRoot, boundedFile)
	if err != nil || scope != boundedFolder || kind != EditorScopeFolder {
		t.Fatalf("scope escaped workspace = %q, %q, %v", scope, kind, err)
	}
}

func TestNotebookEntrypoint(t *testing.T) {
	root := t.TempDir()
	pages := filepath.Join(root, "pages")
	if err := os.MkdirAll(pages, 0755); err != nil {
		t.Fatal(err)
	}
	entrypoint := filepath.Join(pages, "start.typ")
	if err := os.WriteFile(entrypoint, nil, 0644); err != nil {
		t.Fatal(err)
	}
	manifest := []byte("[package]\nentrypoint = \"pages/start.typ\"\n")
	if err := os.WriteFile(filepath.Join(root, "typst.toml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}
	got, err := NotebookEntrypoint(root)
	if err != nil || got != entrypoint {
		t.Fatalf("manifest entrypoint = %q, %v; want %q", got, err, entrypoint)
	}

	outside := filepath.Join(filepath.Dir(root), "outside.typ")
	if err := os.WriteFile(outside, nil, 0644); err != nil {
		t.Fatal(err)
	}
	main := filepath.Join(root, "main.typ")
	if err := os.WriteFile(main, nil, 0644); err != nil {
		t.Fatal(err)
	}
	manifest = []byte("[package]\nentrypoint = \"../outside.typ\"\n")
	if err := os.WriteFile(filepath.Join(root, "typst.toml"), manifest, 0644); err != nil {
		t.Fatal(err)
	}
	got, err = NotebookEntrypoint(root)
	if err != nil || got != main {
		t.Fatalf("contained fallback = %q, %v; want %q", got, err, main)
	}

	if err := os.Remove(main); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(entrypoint); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "typst.toml")); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(root, "alpha.typ")
	if err := os.WriteFile(first, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "zeta.typ"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	got, err = NotebookEntrypoint(root)
	if err != nil || got != first {
		t.Fatalf("first Typst fallback = %q, %v; want %q", got, err, first)
	}

	if err := os.Remove(first); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "zeta.typ")); err != nil {
		t.Fatal(err)
	}
	if _, err := NotebookEntrypoint(root); err == nil {
		t.Fatal("notebook without a Typst file resolved an entrypoint")
	}
}

func TestContextualTreeKeepsClassicExpansionIndependent(t *testing.T) {
	root := t.TempDir()
	scope := filepath.Join(root, "Project")
	if err := os.MkdirAll(scope, 0755); err != nil {
		t.Fatal(err)
	}
	pages := filepath.Join(scope, "pages")
	if err := os.Mkdir(pages, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(pages, "main.typ")
	if err := os.WriteFile(file, nil, 0644); err != nil {
		t.Fatal(err)
	}
	rootNode, err := explorer.NewFileTree(root)
	if err != nil {
		t.Fatal(err)
	}
	scopeNode, err := explorer.NewFileTree(scope)
	if err != nil {
		t.Fatal(err)
	}
	classic := NewTreeView(rootNode)
	contextual := NewTreeView(scopeNode)
	contextual.HideRoot = true
	defer classic.Close()
	defer contextual.Close()

	if !contextual.GetState(scope).Expanded {
		t.Fatal("contextual root is not expanded")
	}
	if classic.GetState(scope).Expanded {
		t.Fatal("contextual expansion mutated Classic tree state")
	}
	if err := contextual.SelectPath(file); err != nil {
		t.Fatal(err)
	}
	contextual.Rebuild()
	if len(contextual.visibleNodes) != 2 || contextual.visibleNodes[0].Node.Path != pages || contextual.visibleNodes[1].Node.Path != file || contextual.visibleNodes[1].Depth != 1 {
		t.Fatalf("contextual rows = %#v", contextual.visibleNodes)
	}
	if contextual.selectedNode == nil || contextual.selectedNode.Path != file {
		t.Fatalf("selected node = %#v", contextual.selectedNode)
	}
	var selectedFolder string
	classic.OnFolderSelectedFunc = func(node *FileNode) { selectedFolder = node.Path }
	classic.OnSelect(scopeNode)
	if selectedFolder != scope {
		t.Fatalf("selected folder = %q, want %q", selectedFolder, scope)
	}
}
