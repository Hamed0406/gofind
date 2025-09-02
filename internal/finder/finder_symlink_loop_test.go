//go:build !windows

package finder

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFollowSymlink_Loop_NoHang(t *testing.T) {
	td := t.TempDir()

	// real tree
	real := filepath.Join(td, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(real, "x.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// symlink dir -> its parent (creates a potential cycle)
	link := filepath.Join(td, "link")
	if err := os.Symlink(td, link); err != nil {
		t.Skipf("symlink creation failed (permissions?): %v", err)
	}

	cfg := Config{
		Root:           td,
		Concurrency:    runtime.GOMAXPROCS(0),
		OutputFormat:   OutputNDJSON,
		FollowSymlinks: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Just ensure it returns (no hang) and doesn’t error.
	err := Run(ctx, os.Stdout, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
