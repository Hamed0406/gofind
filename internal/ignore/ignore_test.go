package ignore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hamed0406/gofind/internal/ignore"
)

func TestMatcher_BasicPatterns(t *testing.T) {
	tmp := t.TempDir()

	// fake repo root
	if err := os.Mkdir(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// create .gitignore
	content := "node_modules/\n*.log\n"
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ignore.New(ignore.Config{
		StartPath:        tmp,
		RespectGitignore: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !m.Match(filepath.Join(tmp, "node_modules"), true) {
		t.Errorf("expected node_modules dir to be ignored")
	}
	if !m.Match(filepath.Join(tmp, "debug.log"), false) {
		t.Errorf("expected *.log to be ignored")
	}
	if m.Match(filepath.Join(tmp, "main.go"), false) {
		t.Errorf("did not expect main.go to be ignored")
	}
}
