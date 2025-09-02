//go:build !windows

package finder

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestHiddenUnixDotfiles(t *testing.T) {
	td := t.TempDir()
	if err := os.WriteFile(filepath.Join(td, "visible.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(td, ".hidden.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(include bool) []string {
		var out bytes.Buffer
		cfg := Config{
			Root:          td,
			IncludeHidden: include,
			OutputFormat:  OutputJSON,
			Concurrency:   2,
		}
		if err := Run(context.Background(), &out, cfg); err != nil {
			t.Fatalf("run: %v", err)
		}
		var arr []Entry
		if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
			t.Fatalf("decode: %v\njson: %s", err, out.String())
		}
		names := make([]string, 0, len(arr))
		for _, e := range arr {
			names = append(names, filepath.Base(e.Path))
		}
		sort.Strings(names)
		return names
	}

	got := run(false)
	for _, n := range got {
		if n[0] == '.' {
			t.Fatalf("dotfile should be excluded when IncludeHidden=false; got %v", got)
		}
	}

	got = run(true)
	if len(got) != 2 {
		t.Fatalf("expected both files when IncludeHidden=true; got %v", got)
	}
}
