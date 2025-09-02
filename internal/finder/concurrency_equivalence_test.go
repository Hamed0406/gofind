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

func TestResultsSameWithDifferentConcurrency(t *testing.T) {
	td := t.TempDir()
	for i := 0; i < 20; i++ {
		p := filepath.Join(td, "a", "b", "c", "f", "g")
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "file_"+fmtInt(i)+".txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	getPaths := func(conc int) []string {
		var out bytes.Buffer
		cfg := Config{
			Root:         td,
			OutputFormat: OutputJSON,
			Concurrency:  conc,
			MaxDepth:     -1,
		}
		if err := Run(context.Background(), &out, cfg); err != nil {
			t.Fatalf("run: %v", err)
		}
		var arr []Entry
		if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
			t.Fatalf("json: %v", err)
		}
		var paths []string
		for _, e := range arr {
			paths = append(paths, e.Path)
		}
		sort.Strings(paths)
		return paths
	}

	a := getPaths(1)
	b := getPaths(8)
	if len(a) != len(b) {
		t.Fatalf("different counts: conc1=%d conc8=%d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("mismatch at %d:\n1: %q\n8: %q", i, a[i], b[i])
		}
	}
}

func fmtInt(n int) string {
	// small helper to avoid strconv import in this file
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
