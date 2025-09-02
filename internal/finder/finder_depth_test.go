package finder

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
	"time"
)

func mk(t *testing.T, base string, rel string, size int, mod time.Time) string {
	t.Helper()
	p := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, bytes.Repeat([]byte("x"), size), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !mod.IsZero() {
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
	return p
}

func collectJSON(t *testing.T, buf *bytes.Buffer) []Entry {
	t.Helper()
	var arr []Entry
	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	if err := dec.Decode(&arr); err != nil {
		t.Fatalf("decode json: %v\njson: %s", err, buf.String())
	}
	return arr
}

func TestMaxDepth(t *testing.T) {
	td := t.TempDir()
	// depth 0 (root): a.txt
	// depth 1:       sub/b.txt
	// depth 2:       sub/deeper/c.txt
	mk(t, td, "a.txt", 1, time.Now())
	mk(t, td, "sub/b.txt", 1, time.Now())
	mk(t, td, "sub/deeper/c.txt", 1, time.Now())

	run := func(maxDepth int) []string {
		var out bytes.Buffer
		cfg := Config{
			Root:          td,
			MaxDepth:      maxDepth,
			OutputFormat:  OutputJSON,
			IncludeHidden: false,
			Concurrency:   4,
		}
		if err := Run(context.Background(), &out, cfg); err != nil {
			t.Fatalf("run: %v", err)
		}
		entries := collectJSON(t, &out)
		var paths []string
		for _, e := range entries {
			paths = append(paths, e.Path)
		}
		sort.Strings(paths)
		return paths
	}

	got0 := run(0)
	if len(got0) == 0 {
		t.Fatalf("expected at least root-level entries")
	}
	for _, p := range got0 {
		if filepath.Dir(p) != td {
			t.Fatalf("max-depth=0 should only include direct children of root; got %q", p)
		}
	}

	got1 := run(1)
	foundDeeper := false
	for _, p := range got1 {
		if filepath.Base(filepath.Dir(p)) == "deeper" {
			foundDeeper = true
		}
	}
	if foundDeeper {
		t.Fatalf("max-depth=1 should not include depth-2 entries")
	}

	gotAll := run(-1)
	var hasC bool
	for _, p := range gotAll {
		if filepath.Base(p) == "c.txt" {
			hasC = true
			break
		}
	}
	if !hasC {
		t.Fatalf("max-depth=-1 should include deepest entries")
	}
}

func TestFilters_Size_Time_Name_Ext(t *testing.T) {
	td := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	newer := time.Now().Add(-1 * time.Hour)

	mk(t, td, "keep.go", 2000, newer)  // keep by size, name, ext, time
	mk(t, td, "skip.txt", 2000, newer) // wrong ext
	mk(t, td, "small.go", 10, newer)   // too small
	mk(t, td, "old.go", 2000, old)     // too old
	mk(t, td, "name.md", 2000, newer)  // name regex won't match

	var out bytes.Buffer
	cfg := Config{
		Root:         td,
		Extensions:   map[string]bool{".go": true},
		NameRegex:    regexp.MustCompile(`keep|old|small`),
		MinSize:      1000,
		After:        time.Now().Add(-24 * time.Hour),
		OutputFormat: OutputJSON,
		Concurrency:  2,
	}
	if err := Run(context.Background(), &out, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	entries := collectJSON(t, &out)
	if len(entries) != 1 || filepath.Base(entries[0].Path) != "keep.go" {
		t.Fatalf("expected only keep.go, got: %+v", entries)
	}
}
