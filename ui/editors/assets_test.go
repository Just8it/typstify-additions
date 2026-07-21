package editors

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"looz.ws/typstify/typst"
)

func TestPasteAssets(t *testing.T) {
	version := typst.Version{Minor: 14}

	t.Run("copies without overwriting", func(t *testing.T) {
		base := t.TempDir()
		root := filepath.Join(base, "project")
		document := filepath.Join(root, "chapters", "main.typ")
		first := filepath.Join(base, "first", "diagram.png")
		second := filepath.Join(base, "second", "diagram.png")
		writeTestFile(t, document, "")
		writeTestFile(t, first, "first")
		writeTestFile(t, second, "second")

		markup, _, err := importAsset(root, document, first, "", version)
		if err != nil {
			t.Fatal(err)
		}
		if markup != `#image("../images/diagram.png")` {
			t.Fatalf("unexpected first markup: %s", markup)
		}
		markup, _, err = importAsset(root, document, second, "", version)
		if err != nil {
			t.Fatal(err)
		}
		if markup != `#image("../images/diagram-2.png")` {
			t.Fatalf("unexpected second markup: %s", markup)
		}
		assertTestFile(t, filepath.Join(root, "images", "diagram.png"), "first")
		assertTestFile(t, filepath.Join(root, "images", "diagram-2.png"), "second")
	})

	t.Run("promotes a standalone document", func(t *testing.T) {
		base := t.TempDir()
		document := filepath.Join(base, "Lecture 01.typ")
		source := filepath.Join(base, "source", "diagram.png")
		writeTestFile(t, document, "Hello\r\nWorld")
		writeTestFile(t, source, "image")

		root, newDocument, err := promoteStandalone(document, source, 6, 6, version)
		if err != nil {
			t.Fatal(err)
		}
		if root != filepath.Join(base, "Lecture 01") || newDocument != filepath.Join(root, "Lecture 01.typ") {
			t.Fatalf("unexpected promoted paths: %s, %s", root, newDocument)
		}
		if _, err := os.Stat(document); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("old document still exists: %v", err)
		}
		assertTestFile(t, newDocument, "Hello\r\n#image(\"images/diagram.png\")World")
		assertTestFile(t, filepath.Join(root, "images", "diagram.png"), "image")
		assertTestFile(t, source, "image")

		blocked := filepath.Join(base, "Blocked.typ")
		writeTestFile(t, blocked, "safe")
		if err := os.Mkdir(filepath.Join(base, "Blocked"), 0755); err != nil {
			t.Fatal(err)
		}
		if _, _, err := promoteStandalone(blocked, source, 4, 4, version); err == nil {
			t.Fatal("expected collision refusal")
		}
		assertTestFile(t, blocked, "safe")
	})

	t.Run("adds the legacy PDF import once", func(t *testing.T) {
		base := t.TempDir()
		document := filepath.Join(base, "main.typ")
		source := filepath.Join(base, "paper.pdf")
		writeTestFile(t, document, "")
		writeTestFile(t, source, "pdf")

		markup, header, err := importAsset(base, document, source, "", typst.Version{Minor: 13})
		if err != nil {
			t.Fatal(err)
		}
		if markup != `#muchpdf(read("pdf/paper.pdf", encoding: none))` || header != muchPDFImport {
			t.Fatalf("unexpected legacy PDF output: %q %q", header, markup)
		}
		_, header, err = importAsset(base, document, source, muchPDFImport, typst.Version{Minor: 13})
		if err != nil {
			t.Fatal(err)
		}
		if header != "" {
			t.Fatalf("duplicate import: %q", header)
		}
	})

	t.Run("maximizes native PDFs", func(t *testing.T) {
		base := t.TempDir()
		document := filepath.Join(base, "main.typ")
		source := filepath.Join(base, "paper.pdf")
		writeTestFile(t, document, "")
		writeTestFile(t, source, "pdf")

		markup, _, err := importAsset(base, document, source, "", version)
		if err != nil {
			t.Fatal(err)
		}
		want := "#image(\n  \"pdf/paper.pdf\",\n  width: 100%,\n  page: 1,\n  fit: \"contain\",\n)"
		if markup != want {
			t.Fatalf("unexpected native PDF markup: %q", markup)
		}
	})
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertTestFile(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("%s: got %q, want %q", path, content, want)
	}
}
