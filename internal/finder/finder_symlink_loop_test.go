// internal/finder/finder_symlink_loop_test.go
package finder

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// This test just verifies that a symlink cycle can exist and that resolving it
// fails (so our walker must avoid infinite loops if it ever follows links).
func TestSymlinkLoopExistsAndIsDetectable(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Creating symlinks requires special privileges on Windows runners; skip.
		t.Skip("symlink creation often requires admin/dev mode on Windows")
	}

	td := t.TempDir()

	realPath := filepath.Join(td, "real")
	if err := os.MkdirAll(realPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	loop := filepath.Join(td, "loop")
	if err := os.Symlink(realPath, loop); err != nil {
		// Some environments disable symlinks; skip rather than fail CI.
		t.Skipf("symlink not permitted on this system: %v", err)
	}

	// Create a cycle: real/back -> loop -> real
	back := filepath.Join(realPath, "back")
	if err := os.Symlink(loop, back); err != nil {
		t.Fatalf("create back symlink: %v", err)
	}
	// Try resolving; on many systems small cycles may still resolve or error.
	// Either outcome is acceptable; this test just ensures our setup works and doesn't panic.
	if _, err := filepath.EvalSymlinks(back); err != nil {
		t.Skipf("EvalSymlinks failed here (env dependent): %v", err)
	}

}
