package filetree

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/oligo/gioview/explorer"
	"github.com/typstify/tpix-cli/bundler"
)

type FocusedEntry struct {
	Node       *FileNode
	IsNotebook bool
}

type Breadcrumb struct {
	Name string
	Path string
}

type EditorScopeKind string

const (
	EditorScopeProject EditorScopeKind = "project"
	EditorScopeFolder  EditorScopeKind = "folder"
)

type FocusedBrowser struct {
	root    string
	current string
	node    *FileNode
}

func NewFocusedBrowser(root, preferred string) (*FocusedBrowser, error) {
	browser := &FocusedBrowser{}
	if err := browser.SetWorkspace(root, preferred); err != nil {
		return nil, err
	}
	return browser, nil
}

func (b *FocusedBrowser) Root() string    { return b.root }
func (b *FocusedBrowser) Current() string { return b.current }
func (b *FocusedBrowser) Node() *FileNode { return b.node }

func (b *FocusedBrowser) SetWorkspace(root, preferred string) error {
	root, err := absoluteDirectory(root)
	if err != nil {
		return err
	}

	b.root = root
	b.current = closestExistingDirectory(root, preferred)
	return b.refreshNode()
}

func (b *FocusedBrowser) Enter(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	path = filepath.Clean(path)
	if !PathWithin(b.root, path) {
		return fmt.Errorf("path is outside workspace: %s", path)
	}
	if _, err := absoluteDirectory(path); err != nil {
		return err
	}

	b.current = path
	return b.refreshNode()
}

func (b *FocusedBrowser) Up() error {
	if b.current == b.root {
		return nil
	}
	return b.Enter(filepath.Dir(b.current))
}

func (b *FocusedBrowser) Breadcrumbs() []Breadcrumb {
	crumbs := []Breadcrumb{{Name: filepath.Base(b.root), Path: b.root}}
	if b.current == b.root {
		return crumbs
	}

	rel, err := filepath.Rel(b.root, b.current)
	if err != nil {
		return crumbs
	}
	path := b.root
	for _, name := range strings.Split(rel, string(os.PathSeparator)) {
		path = filepath.Join(path, name)
		crumbs = append(crumbs, Breadcrumb{Name: name, Path: path})
	}
	return crumbs
}

func (b *FocusedBrowser) Entries() ([]FocusedEntry, error) {
	current := closestExistingDirectory(b.root, b.current)
	if current != b.current || b.node == nil {
		b.current = current
		if err := b.refreshNode(); err != nil {
			return nil, err
		}
	} else if err := b.node.Refresh(nil); err != nil {
		return nil, err
	}

	entries := make([]FocusedEntry, 0, len(b.node.Children()))
	for _, node := range b.node.Children() {
		if node.IsDir() && strings.HasPrefix(node.Name(), ".") {
			continue
		}
		entries = append(entries, FocusedEntry{Node: node, IsNotebook: isNotebook(node)})
	}
	slices.SortFunc(entries, func(a, b FocusedEntry) int {
		if a.Node.IsDir() != b.Node.IsDir() {
			if a.Node.IsDir() {
				return -1
			}
			return 1
		}
		if cmp := strings.Compare(strings.ToLower(a.Node.Name()), strings.ToLower(b.Node.Name())); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.Node.Name(), b.Node.Name())
	})
	return entries, nil
}

func (b *FocusedBrowser) refreshNode() error {
	node, err := explorer.NewFileTree(b.current)
	if err != nil {
		return err
	}
	if err := node.Refresh(nil); err != nil {
		return err
	}
	b.node = node
	return nil
}

func isNotebook(node *FileNode) bool {
	if !node.IsDir() {
		return false
	}
	info, err := os.Stat(filepath.Join(node.Path, "typst.toml"))
	return err == nil && !info.IsDir()
}

func ResolveEditorScope(root, file string) (string, EditorScopeKind, error) {
	root, err := absoluteDirectory(root)
	if err != nil {
		return "", "", err
	}
	file, err = filepath.Abs(file)
	if err != nil {
		return "", "", err
	}
	file = filepath.Clean(file)
	if !PathWithin(root, file) {
		return "", "", fmt.Errorf("path is outside workspace: %s", file)
	}
	info, err := os.Stat(file)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return "", "", fmt.Errorf("not a file: %s", file)
	}

	folder := filepath.Dir(file)
	for current := folder; PathWithin(root, current); current = filepath.Dir(current) {
		if marker, err := os.Stat(filepath.Join(current, "typst.toml")); err == nil && !marker.IsDir() {
			return current, EditorScopeProject, nil
		}
		if current == root {
			break
		}
	}
	return folder, EditorScopeFolder, nil
}

func NotebookEntrypoint(path string) (string, error) {
	path, err := absoluteDirectory(path)
	if err != nil {
		return "", err
	}
	manifestData, _ := os.ReadFile(filepath.Join(path, "typst.toml"))
	var manifest bundler.Manifest
	if bundler.DecodeBytes(manifestData, &manifest) == nil && manifest.Package != nil {
		if entrypoint, ok := notebookTypFile(path, manifest.Package.Entrypoint); ok {
			return entrypoint, nil
		}
	}
	if entrypoint, ok := notebookTypFile(path, "main.typ"); ok {
		return entrypoint, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".typ") {
			return filepath.Join(path, entry.Name()), nil
		}
	}
	return "", fmt.Errorf("notebook has no Typst entry file: %s", path)
}

func notebookTypFile(root, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	path, err := filepath.Abs(filepath.Join(root, name))
	if err != nil || !PathWithin(root, path) {
		return "", false
	}
	info, err := os.Stat(path)
	return path, err == nil && !info.IsDir() && strings.EqualFold(filepath.Ext(path), ".typ")
}

func absoluteDirectory(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
	}
	return path, nil
}

func closestExistingDirectory(root, path string) string {
	if path == "" {
		return root
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return root
	}
	path = filepath.Clean(path)
	if !PathWithin(root, path) {
		return root
	}

	for PathWithin(root, path) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
		if path == root {
			break
		}
		path = filepath.Dir(path)
	}
	return root
}

func PathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
