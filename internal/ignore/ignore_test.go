// internal/ignore/ignore_test.go
package ignore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamed0406/gofind/internal/ignore"
)

func TestMatcher_BasicPatterns(t *testing.T) {
	td := t.TempDir()

	// Layout:
	//   /node_modules/pkg/index.js
	//   /build/out.bin
	//   /keep/file.tmp
	//   /keep/readme.md
	nodeMod := filepath.Join(td, "node_modules", "pkg")
	if err := os.MkdirAll(nodeMod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeMod, "index.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(td, "build"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "build", "out.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(td, "keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "keep", "file.tmp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, "keep", "readme.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Ignore node_modules directories and *.tmp files, but NOT build/ by name here.
	cfg := ignore.Config{
		Root:     td,
		Patterns: []string{"node_modules/", "*.tmp"},
		Enabled:  true,
	}
	m, err := ignore.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// node_modules (dir) should be ignored
	if !m.Match(filepath.Join(td, "node_modules"), true) {
		t.Fatalf("expected node_modules dir to be ignored")
	}
	// file under node_modules should be ignored
	if !m.Match(filepath.Join(td, "node_modules", "pkg", "index.js"), false) {
		t.Fatalf("expected file under node_modules to be ignored")
	}
	// *.tmp should be ignored
	if !m.Match(filepath.Join(td, "keep", "file.tmp"), false) {
		t.Fatalf("expected *.tmp file to be ignored")
	}
	// readme.md should NOT be ignored
	if m.Match(filepath.Join(td, "keep", "readme.md"), false) {
		t.Fatalf("did not expect readme.md to be ignored")
	}
	// build/ was not in patterns → NOT ignored
	if m.Match(filepath.Join(td, "build"), true) {
		t.Fatalf("did not expect build/ to be ignored")
	}
}

func TestMatcher_Disabled(t *testing.T) {
	td := t.TempDir()
	cfg := ignore.Config{
		Root:     td,
		Patterns: []string{"node_modules/", "*.tmp"},
		Enabled:  false, // disabled
	}
	m, err := ignore.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if m.Match(filepath.Join(td, "node_modules"), true) {
		t.Fatalf("with Enabled=false, nothing should match")
	}
}
