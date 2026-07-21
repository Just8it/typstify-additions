package editors

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		assertTestFile(t, filepath.Join(root, "typst.toml"), "[package]\nname = \"notebook\"\nversion = \"0.1.0\"\nentrypoint = \"Lecture 01.typ\"\n")
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

	t.Run("uses the nearest notebook in Student mode", func(t *testing.T) {
		library := t.TempDir()
		outer := filepath.Join(library, "Course")
		inner := filepath.Join(outer, "Week 1")
		document := filepath.Join(inner, "notes.typ")
		loose := filepath.Join(library, "Loose", "notes.typ")
		writeTestFile(t, filepath.Join(outer, "typst.toml"), "")
		writeTestFile(t, filepath.Join(inner, "typst.toml"), "")
		writeTestFile(t, document, "")
		writeTestFile(t, loose, "")

		if root, ok := assetProjectRoot(library, document, true); !ok || root != inner {
			t.Fatalf("got %q, %v; want nested notebook %q", root, ok, inner)
		}
		if root, ok := assetProjectRoot(library, loose, true); ok || root != filepath.Dir(loose) {
			t.Fatalf("got %q, %v; want loose document folder", root, ok)
		}
		if root, ok := assetProjectRoot(library, loose, false); !ok || root != library {
			t.Fatalf("got %q, %v; want Classic workspace %q", root, ok, library)
		}
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

	t.Run("imports every native PDF page", func(t *testing.T) {
		base := t.TempDir()
		document := filepath.Join(base, "main.typ")
		source := filepath.Join(base, "paper.pdf")
		writeTestFile(t, document, "")
		writeTestPDF(t, source, 3)

		markup, header, err := importAsset(base, document, source, "", version)
		if err != nil {
			t.Fatal(err)
		}
		want := "#typstify-pdf(\n  \"pdf/paper.pdf\",\n  3, // detected page count\n  pages: auto, // all; or (first, last); or one page\n  width: 100%,\n  fit: \"contain\",\n)"
		if markup != want {
			t.Fatalf("unexpected native PDF markup: %q", markup)
		}
		if header != nativePDFHelper {
			t.Fatalf("missing native PDF helper: %q", header)
		}
		_, header, err = importAsset(base, document, source, nativePDFHelper, version)
		if err != nil {
			t.Fatal(err)
		}
		if header != "" {
			t.Fatalf("duplicate helper: %q", header)
		}
	})

	t.Run("rejects invalid native PDFs before copying", func(t *testing.T) {
		base := t.TempDir()
		document := filepath.Join(base, "main.typ")
		source := filepath.Join(base, "broken.pdf")
		writeTestFile(t, document, "")
		writeTestFile(t, source, "not a PDF")

		if _, _, err := importAsset(base, document, source, "", version); err == nil {
			t.Fatal("expected invalid PDF error")
		}
		if _, err := os.Stat(filepath.Join(base, "pdf")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("invalid PDF created an asset folder: %v", err)
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

func writeTestPDF(t *testing.T, path string, pages int) {
	t.Helper()
	pageObjects := make([]string, pages)
	pageRefs := make([]string, pages)
	for i := range pages {
		pageObjects[i] = "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] >>"
		pageRefs[i] = fmt.Sprintf("%d 0 R", i+3)
	}
	objects := append([]string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(pageRefs, " "), pages),
	}, pageObjects...)

	var contents bytes.Buffer
	contents.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects))
	for i, object := range objects {
		offsets[i] = contents.Len()
		fmt.Fprintf(&contents, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := contents.Len()
	fmt.Fprintf(&contents, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&contents, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&contents, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	writeTestFile(t, path, contents.String())
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
