package editors

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"looz.ws/typstify/typst"
)

const muchPDFImport = "#import \"@preview/muchpdf:0.1.2\": muchpdf\n"

func assetKind(path string, version typst.Version) (folder string, pdf, supported bool, err error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		return "images", false, true, nil
	case ".pdf":
		if !version.AtLeast(0, 13, 0) {
			return "", true, true, fmt.Errorf("pasting PDFs requires Typst 0.13 or newer")
		}
		return "pdf", true, true, nil
	default:
		return "", false, false, nil
	}
}

func importAsset(root, document, source, content string, version typst.Version) (markup, header string, err error) {
	folder, pdf, supported, err := assetKind(source, version)
	if err != nil {
		return "", "", err
	}
	if !supported {
		return "", "", fmt.Errorf("unsupported pasted file: %s", filepath.Base(source))
	}
	if !pathWithin(root, document) {
		return "", "", fmt.Errorf("document is outside the project")
	}

	info, err := os.Stat(source)
	if err != nil {
		return "", "", fmt.Errorf("read pasted file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("pasted item is not a regular file")
	}

	destinationDir := filepath.Join(root, folder)
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		return "", "", fmt.Errorf("create asset folder: %w", err)
	}
	destination, err := copyUnique(source, destinationDir, info)
	if err != nil {
		return "", "", err
	}

	relative, err := filepath.Rel(filepath.Dir(document), destination)
	if err != nil {
		return "", "", fmt.Errorf("make asset path relative: %w", err)
	}
	path := strconv.Quote(filepath.ToSlash(relative))
	if !pdf {
		return "#image(" + path + ")", "", nil
	}
	if version.AtLeast(0, 14, 0) {
		return "#image(\n  " + path + ",\n  width: 100%,\n  page: 1,\n  fit: \"contain\",\n)", "", nil
	}

	if !strings.Contains(content, strings.TrimSpace(muchPDFImport)) {
		header = muchPDFImport
	}
	return "#muchpdf(read(" + path + ", encoding: none))", header, nil
}

func copyUnique(source, destinationDir string, sourceInfo os.FileInfo) (string, error) {
	ext := filepath.Ext(source)
	stem := strings.TrimSuffix(filepath.Base(source), ext)
	for number := 1; ; number++ {
		name := stem + ext
		if number > 1 {
			name = stem + "-" + strconv.Itoa(number) + ext
		}
		destination := filepath.Join(destinationDir, name)

		if destinationInfo, err := os.Stat(destination); err == nil {
			if os.SameFile(sourceInfo, destinationInfo) {
				return destination, nil
			}
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect asset destination: %w", err)
		}

		output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, sourceInfo.Mode().Perm())
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create asset: %w", err)
		}

		input, err := os.Open(source)
		if err == nil {
			_, err = io.Copy(output, input)
			input.Close()
		}
		closeErr := output.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			os.Remove(destination)
			return "", fmt.Errorf("copy asset: %w", err)
		}
		return destination, nil
	}
}

func applyPaste(content string, start, end int, header, markup string) (string, int, error) {
	runes := []rune(content)
	if start > end {
		start, end = end, start
	}
	if start < 0 || end > len(runes) {
		return "", 0, fmt.Errorf("selection is outside the document")
	}
	result := header + string(runes[:start]) + markup + string(runes[end:])
	caret := len([]rune(header)) + start + len([]rune(markup))
	return result, caret, nil
}

func promoteStandalone(document, source string, start, end int, version typst.Version) (root, newDocument string, err error) {
	document, err = filepath.Abs(document)
	if err != nil {
		return "", "", err
	}
	parent := filepath.Dir(document)
	stem := strings.TrimSuffix(filepath.Base(document), filepath.Ext(document))
	root = filepath.Join(parent, stem)
	if _, err := os.Lstat(root); err == nil {
		return "", "", fmt.Errorf("cannot create project: %s already exists", root)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("inspect project destination: %w", err)
	}

	info, err := os.Stat(document)
	if err != nil {
		return "", "", fmt.Errorf("read standalone document: %w", err)
	}
	content, err := os.ReadFile(document)
	if err != nil {
		return "", "", fmt.Errorf("read standalone document: %w", err)
	}
	usesCRLF := strings.Contains(string(content), "\r\n")
	normalized := strings.ReplaceAll(strings.ReplaceAll(string(content), "\r\n", "\n"), "\r", "\n")

	staging, err := os.MkdirTemp(parent, "."+stem+"-")
	if err != nil {
		return "", "", fmt.Errorf("create project: %w", err)
	}
	defer os.RemoveAll(staging)

	stagingDocument := filepath.Join(staging, filepath.Base(document))
	markup, header, err := importAsset(staging, stagingDocument, source, normalized, version)
	if err != nil {
		return "", "", err
	}
	updated, _, err := applyPaste(normalized, start, end, header, markup)
	if err != nil {
		return "", "", err
	}
	if usesCRLF {
		updated = strings.ReplaceAll(updated, "\n", "\r\n")
	}
	if err := os.WriteFile(stagingDocument, []byte(updated), info.Mode().Perm()); err != nil {
		return "", "", fmt.Errorf("write project document: %w", err)
	}
	if err := os.Rename(staging, root); err != nil {
		return "", "", fmt.Errorf("publish project: %w", err)
	}
	if err := os.Remove(document); err != nil {
		if rollbackErr := os.RemoveAll(root); rollbackErr != nil {
			return "", "", fmt.Errorf("remove old document: %v (rollback also failed: %v)", err, rollbackErr)
		}
		return "", "", fmt.Errorf("remove old document: %w", err)
	}

	return root, filepath.Join(root, filepath.Base(document)), nil
}

func pathWithin(root, path string) bool {
	root, rootErr := filepath.Abs(root)
	path, pathErr := filepath.Abs(path)
	if rootErr != nil || pathErr != nil {
		return false
	}
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
