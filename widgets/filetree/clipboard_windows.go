//go:build windows

package filetree

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const cfHDrop = 15

var (
	user32            = windows.NewLazySystemDLL("user32.dll")
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	openClipboard     = user32.NewProc("OpenClipboard")
	closeClipboard    = user32.NewProc("CloseClipboard")
	getClipboardData  = user32.NewProc("GetClipboardData")
	dragQueryFileWide = shell32.NewProc("DragQueryFileW")
)

// ReadClipboardFiles returns files copied by Windows Explorer.
func ReadClipboardFiles() []string {
	ok, _, _ := openClipboard.Call(0)
	if ok == 0 {
		return nil
	}
	defer closeClipboard.Call()

	hdrop, _, _ := getClipboardData.Call(cfHDrop)
	if hdrop == 0 {
		return nil
	}

	count, _, _ := dragQueryFileWide.Call(hdrop, 0xffffffff, 0, 0)
	files := make([]string, 0, int(count))
	for i := uintptr(0); i < count; i++ {
		length, _, _ := dragQueryFileWide.Call(hdrop, i, 0, 0)
		if length == 0 {
			continue
		}
		buf := make([]uint16, int(length)+1)
		dragQueryFileWide.Call(hdrop, i, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		files = append(files, windows.UTF16ToString(buf))
	}
	return files
}
