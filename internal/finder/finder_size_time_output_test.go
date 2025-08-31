package finder_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hamed0406/gofind/internal/finder"
)

func TestSizeTimeAndOutput(t *testing.T) {
	tmp := t.TempDir()
	_ = os.Mkdir(filepath.Join(tmp, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte(""), 0o644)

	// create files with different sizes and mtimes
	small := filepath.Join(tmp, "small.txt")
	mid := filepath.Join(tmp, "mid.txt")
	large := filepath.Join(tmp, "large.txt")

	mustWrite := func(p string, size int) {
		data := bytes.Repeat([]byte("x"), size)
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(small, 10*1024)     // 10K
	mustWrite(mid, 2*1024*1024)   // 2M
	mustWrite(large, 5*1024*1024) // 5M

	// set mtimes: small = now-10d, mid = now-2d, large = now-1h
	now := time.Now()
	if err := os.Chtimes(small, now.Add(-10*24*time.Hour), now.Add(-10*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(mid, now.Add(-48*time.Hour), now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(large, now.Add(-1*time.Hour), now.Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}

	// larger than 1M should include mid, large
	{
		var buf bytes.Buffer
		cfg := finder.Config{
			Path:   tmp,
			Output: "path",
			Larger: "1M",
		}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		out := buf.String()
		if !strings.Contains(out, "mid.txt") || !strings.Contains(out, "large.txt") {
			t.Fatalf("--larger filter failed, got:\n%s", out)
		}
		if strings.Contains(out, "small.txt") {
			t.Fatalf("--larger should exclude small.txt")
		}
	}

	// smaller than 3M should include small, mid
	{
		var buf bytes.Buffer
		cfg := finder.Config{
			Path:    tmp,
			Output:  "path",
			Smaller: "3M",
		}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		out := buf.String()
		if !strings.Contains(out, "small.txt") || !strings.Contains(out, "mid.txt") {
			t.Fatalf("--smaller filter failed, got:\n%s", out)
		}
		if strings.Contains(out, "large.txt") {
			t.Fatalf("--smaller should exclude large.txt")
		}
	}

	// since 2d should include mid (2d) and large (1h), exclude small (10d)
	{
		var buf bytes.Buffer
		cfg := finder.Config{
			Path:   tmp,
			Output: "path",
			Since:  "2d",
		}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		out := buf.String()
		if !strings.Contains(out, "mid.txt") || !strings.Contains(out, "large.txt") {
			t.Fatalf("--since filter failed, got:\n%s", out)
		}
		if strings.Contains(out, "small.txt") {
			t.Fatalf("--since should exclude small.txt")
		}
	}

	// JSON output should be a JSON array with objects
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, Output: "json"}
		_, _ = finder.Run(context.Background(), &buf, cfg)

		var arr []finder.Result
		if err := json.Unmarshal(buf.Bytes(), &arr); err != nil {
			t.Fatalf("json unmarshal failed: %v\npayload:\n%s", err, buf.String())
		}
		if len(arr) == 0 {
			t.Fatalf("expected some json results")
		}
	}

	// NDJSON output should be newline-separated JSON objects
	{
		var buf bytes.Buffer
		cfg := finder.Config{Path: tmp, Output: "ndjson"}
		_, _ = finder.Run(context.Background(), &buf, cfg)
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) == 0 {
			t.Fatalf("expected some ndjson lines")
		}
		var r finder.Result
		if err := json.Unmarshal([]byte(lines[0]), &r); err != nil {
			t.Fatalf("ndjson first line invalid json: %v", err)
		}
	}
}
