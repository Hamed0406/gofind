//go:build !windows

package finder

import (
	"io/fs"
	"syscall"
)

// Keep a package-level reference so linters don't mark this as unused on Unix.
// Remove this once the walker calls statFromFileInfo directly.
var _ = statFromFileInfo

// statFromFileInfo extracts inode and device numbers from a FileInfo on Unix.
// Returns ok=false if syscall.Stat_t is not available.
func statFromFileInfo(info fs.FileInfo) (inode, dev uint64, ok bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st == nil {
		return 0, 0, false
	}
	return uint64(st.Ino), uint64(st.Dev), true
}
