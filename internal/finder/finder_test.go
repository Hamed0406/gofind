package finder_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hamed0406/gofind/internal/finder"
)

func TestRun_RespectsGitignore(t *testing.T) {
	tmp := t.TempDir()

	// Create .git and .gitignore
	if err := os.Mkdir(filepath.Join(tmp, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	ignoreContent := "secret/\n*.log\n"
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte(ignoreContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create files
	files := []string{
		"main.go",
		"README.md",
		"debug.log",
		"secret/passwords.txt",
	}
	for _, f := range files {
		path := filepath.Join(tmp, f)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Run finder
	var buf bytes.Buffer
	cfg := finder.Config{
		Path:             tmp,
		RespectGitignore: true,
	}
	count, err := finder.Run(context.Background(), &buf, cfg)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if count == 0 {
		t.Fatal("expected some files to be listed")
	}
	if strings.Contains(out, "debug.log") {
		t.Errorf("debug.log should have been ignored")
	}
	if strings.Contains(out, "secret") {
		t.Errorf("secret dir should have been ignored")
	}
	if !strings.Contains(out, "main.go") {
		t.Errorf("expected main.go to be listed")
	}
}
