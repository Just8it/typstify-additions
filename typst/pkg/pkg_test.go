package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalPkgsIncludesLocalStylePackage(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "local", "uninotes", "0.1.0")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `[package]
name = "uninotes"
version = "0.1.0"
entrypoint = "lib.typ"
description = "Personal lecture-note style"
`
	if err := os.WriteFile(filepath.Join(dir, "typst.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	packages, err := (&TypstPkgService{packageDir: root}).LocalPkgs()
	if err != nil {
		t.Fatal(err)
	}
	if len(packages) != 1 {
		t.Fatalf("got %d local packages, want 1", len(packages))
	}
	got := packages[0]
	if got.ImportPath() != "@local/uninotes:0.1.0" || got.IsTemplate {
		t.Fatalf("unexpected local package: %#v", got)
	}
}
