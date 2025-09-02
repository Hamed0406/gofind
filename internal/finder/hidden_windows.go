// internal/finder/hidden_windows.go
//go:build windows

package finder

import "syscall"

// Windows hidden detection using file attributes.
func isHidden(path, name string) bool {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return false
	}
	const FILE_ATTRIBUTE_HIDDEN = 0x2
	return attrs&FILE_ATTRIBUTE_HIDDEN != 0
}
