//go:build !darwin && !windows

package filetree

func ReadClipboardFiles() []string {
	return nil
}
