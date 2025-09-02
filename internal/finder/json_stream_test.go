package finder

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestJSONStreamingOutputIsValid(t *testing.T) {
	td := t.TempDir()
	// make a handful of files across a couple of subdirs
	for i := 0; i < 5; i++ {
		dir := filepath.Join(td, "d", "sub")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		fp := filepath.Join(dir, "f"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(fp, []byte(strings.Repeat("x", 10)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	cfg := Config{
		Root:         td,
		OutputFormat: OutputJSON,
		MaxDepth:     -1,
		Concurrency:  8,
	}
	if err := Run(context.Background(), &out, cfg); err != nil {
		t.Fatalf("run: %v", err)
	}

	// Ensure it's a JSON array and decodes into entries
	var arr []Entry
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
		t.Fatalf("unmarshal: %v\njson: %s", err, out.String())
	}
	if len(arr) == 0 {
		t.Fatalf("expected some entries in JSON stream")
	}

	// Basic shape checks
	for _, e := range arr {
		if e.Path == "" || e.Name == "" {
			t.Fatalf("invalid entry: %+v", e)
		}
	}

	// Ensure order doesn't matter; just verify all files are present.
	var names []string
	for _, e := range arr {
		if !e.IsDir {
			names = append(names, e.Name)
		}
	}
	sort.Strings(names)
	if len(names) < 5 {
		t.Fatalf("expected at least 5 files, got %d (%v)", len(names), names)
	}
}
