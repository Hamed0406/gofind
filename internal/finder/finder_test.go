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

func TestRun_Ext_Name_Regex_Type_Hidden(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Mkdir(filepath.Join(tmp, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte(""), 0o644)

	// layout
	mustWrite := func(rel string) {
		p := filepath.Join(tmp, rel)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if strings.HasSuffix(rel, "/") {
			_ = os.MkdirAll(p, 0o755)
		} else {
			_ = os.WriteFile(p, []byte("x"), 0o644)
		}
	}
	mustWrite("src/main.go")
	mustWrite("src/util_test.go")
	mustWrite("docs/readme.md")
	mustWrite(".hidden/secret.txt")
	_ = os.MkdirAll(filepath.Join(tmp, "onlydir"), 0o755)

	// ext filter
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, RespectGitignore: true, Exts: []string{".go"}}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		got := buf.String()
		if !strings.Contains(got, "main.go") || !strings.Contains(got, "util_test.go") {
			t.Fatalf("ext filter failed, got:\n%s", got)
		}
		if strings.Contains(got, "readme.md") {
			t.Fatalf("ext filter should exclude readme.md")
		}
	}

	// name filter (substring)
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, RespectGitignore: true, Name: "readme"}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		if !strings.Contains(buf.String(), "readme.md") {
			t.Fatalf("name filter failed")
		}
	}

	// regex filter
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, RespectGitignore: true, Regex: `^main\.`}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		if !strings.Contains(buf.String(), "main.go") {
			t.Fatalf("regex filter failed")
		}
		if strings.Contains(buf.String(), "util_test.go") {
			t.Fatalf("regex filter should exclude util_test.go")
		}
	}

	// type = d (dirs only)
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, RespectGitignore: true, Type: "d"}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		out := buf.String()
		if strings.Contains(out, "main.go") || strings.Contains(out, "readme.md") {
			t.Fatalf("type=d should not list files")
		}
		// should include at least the start dir subfolders like src, docs, .hidden, onlydir
		if !strings.Contains(out, "src") {
			t.Fatalf("type=d should list directories; got:\n%s", out)
		}
	}

	// hidden false: should not include .hidden content
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, RespectGitignore: true, Hidden: false}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		if strings.Contains(buf.String(), "/.hidden/") {
			t.Fatalf("hidden=false should exclude .hidden")
		}
	}

	// hidden true: include .hidden
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, RespectGitignore: true, Hidden: true}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		if !strings.Contains(buf.String(), "/.hidden/secret.txt") &&
			!strings.Contains(buf.String(), "\\.hidden\\secret.txt") {
			t.Fatalf("hidden=true should include .hidden/secret.txt")
		}
	}
}
