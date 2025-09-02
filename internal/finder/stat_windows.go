//go:build windows

package finder

import "io/fs"

// Windows doesn't expose Unix inode/dev semantics the same way.
// Return ok=false so callers can skip inode/dev-only paths.
//
//nolint:unused // referenced once we wire inode/dev-based logic; keep shim compiled on Unix
func statFromFileInfo(info fs.FileInfo) (inode, dev uint64, ok bool) {
	return 0, 0, false
}
