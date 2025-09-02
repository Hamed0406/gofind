// internal/finder/finder_test.go
package finder

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
	"time"
)

func decodeJSON(t *testing.T, b *bytes.Buffer) []Entry {
	t.Helper()
	var arr []Entry
	if err := json.Unmarshal(b.Bytes(), &arr); err != nil {
		t.Fatalf("json decode: %v\nraw: %s", err, b.String())
	}
	return arr
}

func TestExtFilterAndNameRegex(t *testing.T) {
	td := t.TempDir()
	_ = mkFile(t, td, "keep/alpha.go", 10, time.Now())
	_ = mkFile(t, td, "keep/beta.go", 10, time.Now())
	_ = mkFile(t, td, "skip/readme.md", 10, time.Now())

	var out bytes.Buffer
	cfg := Config{
		Root:         td,
		Extensions:   map[string]bool{".go": true},
		NameRegex:    regexp.MustCompile(`alpha|beta`),
		OutputFormat: OutputJSON,
		MaxDepth:     -1, // allow scanning into keep/
	}
	if err := Run(context.Background(), &out, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	entries := decodeJSON(t, &out)
	var names []string
	for _, e := range entries {
		if !e.IsDir {
			names = append(names, filepath.Base(e.Path))
		}
	}
	sort.Strings(names)
	want := []string{"alpha.go", "beta.go"}
	if len(names) != 2 || names[0] != want[0] || names[1] != want[1] {
		t.Fatalf("want %v, got %v", want, names)
	}
}

func TestSizeAndTimeFilters(t *testing.T) {
	td := t.TempDir()
	old := time.Now().Add(-72 * time.Hour)
	newer := time.Now().Add(-1 * time.Hour)

	_ = mkFile(t, td, "ok.go", 2000, newer)  // keep
	_ = mkFile(t, td, "small.go", 10, newer) // too small
	_ = mkFile(t, td, "old.go", 2000, old)   // too old

	var out bytes.Buffer
	cfg := Config{
		Root:         td,
		Extensions:   map[string]bool{".go": true},
		MinSize:      1000,
		After:        time.Now().Add(-48 * time.Hour),
		OutputFormat: OutputJSON,
	}
	if err := Run(context.Background(), &out, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}
	entries := decodeJSON(t, &out)
	if len(entries) != 1 || filepath.Base(entries[0].Path) != "ok.go" {
		t.Fatalf("expected only ok.go, got %+v", entries)
	}
}
