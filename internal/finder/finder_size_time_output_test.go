// internal/finder/finder_size_time_output_test.go
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

func TestSizeTimeAndNameFilters_JSON(t *testing.T) {
	td := t.TempDir()
	old := time.Now().Add(-72 * time.Hour)
	recent := time.Now().Add(-1 * time.Hour)

	// Files
	pKeep := mkFile(t, td, "keep.go", 2000, recent) // should pass all filters
	_ = mkFile(t, td, "skip_ext.txt", 2000, recent) // wrong ext
	_ = mkFile(t, td, "too_small.go", 10, recent)   // too small
	_ = mkFile(t, td, "too_old.go", 2000, old)      // too old
	_ = mkFile(t, td, "name.md", 2000, recent)      // name regex won't match

	var out bytes.Buffer
	cfg := Config{
		Root:         td,
		Extensions:   map[string]bool{".go": true},
		NameRegex:    regexp.MustCompile(`keep|too_old|too_small`),
		MinSize:      1000,
		After:        time.Now().Add(-48 * time.Hour),
		OutputFormat: OutputJSON,
		Concurrency:  2,
	}
	if err := Run(context.Background(), &out, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	var entries []Entry
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("json decode: %v\nraw: %s", err, out.String())
	}

	// Expect only keep.go
	want := filepath.Base(pKeep)
	if len(entries) != 1 || filepath.Base(entries[0].Path) != want {
		t.Fatalf("expected only %q, got: %+v", want, entries)
	}
}

func TestBeforeTimeFilter(t *testing.T) {
	td := t.TempDir()
	t1 := time.Now().Add(-24 * time.Hour)
	t2 := time.Now().Add(-1 * time.Hour)

	_ = mkFile(t, td, "a.go", 100, t1) // before cutoff -> keep
	_ = mkFile(t, td, "b.go", 100, t2) // after cutoff -> drop

	var out bytes.Buffer
	cfg := Config{
		Root:         td,
		Extensions:   map[string]bool{".go": true},
		Before:       time.Now().Add(-2 * time.Hour),
		OutputFormat: OutputJSON,
	}
	if err := Run(context.Background(), &out, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	var entries []Entry
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("json decode: %v\nraw: %s", err, out.String())
	}
	if len(entries) != 1 || filepath.Base(entries[0].Path) != "a.go" {
		t.Fatalf("expected only a.go, got %+v", entries)
	}
}

func TestNameRegex_NoExtFilter(t *testing.T) {
	td := t.TempDir()
	_ = mkFile(t, td, "alpha.txt", 1, time.Now())
	_ = mkFile(t, td, "beta.md", 1, time.Now())
	_ = mkFile(t, td, "gamma.go", 1, time.Now())

	var out bytes.Buffer
	cfg := Config{
		Root:         td,
		NameRegex:    regexp.MustCompile(`^(alpha|gamma)`),
		OutputFormat: OutputJSON,
	}
	if err := Run(context.Background(), &out, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	var entries []Entry
	if err := json.Unmarshal(out.Bytes(), &entries); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir {
			names = append(names, filepath.Base(e.Path))
		}
	}
	sort.Strings(names)
	want := []string{"alpha.txt", "gamma.go"}
	if len(names) != 2 || names[0] != want[0] || names[1] != want[1] {
		t.Fatalf("want %v, got %v", want, names)
	}
}

// helper
func mkFile(t *testing.T, base, rel string, size int, mod time.Time) string {
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
