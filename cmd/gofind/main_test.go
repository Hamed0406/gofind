// cmd/gofind/main_test.go
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

type cliEntry struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	IsDir   bool      `json:"isDir"`
}

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gofind_testbin")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go build: %v", err)
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("built binary not found: %v", err)
	}
	return bin
}

func mk(t *testing.T, dir, rel string, size int) string {
	t.Helper()
	fp := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(fp, bytes.Repeat([]byte("x"), size), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return fp
}

func TestCLI_VersionFlag(t *testing.T) {
	bin := buildCLI(t)
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v; out=%s", err, string(out))
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		t.Fatalf("expected non-empty version string")
	}
}

func TestCLI_JSON_Array_and_ExtFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skip in short mode")
	}
	bin := buildCLI(t)

	td := t.TempDir()
	mk(t, td, "keep/a.txt", 100)
	mk(t, td, "skip/b.md", 100)
	mk(t, td, "keep/c.txt", 50)

	cmd := exec.Command(bin, "-root", td, "-json", "-ext", ".txt", "-concurrency", "2")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
	}

	var arr []cliEntry
	if err := json.NewDecoder(&out).Decode(&arr); err != nil {
		t.Fatalf("json decode: %v\nraw: %s", err, out.String())
	}
	if len(arr) == 0 {
		t.Fatal("expected some entries")
	}
	for _, e := range arr {
		if !e.IsDir && !strings.EqualFold(filepath.Ext(e.Name), ".txt") {
			t.Fatalf("expected only .txt files, saw %q", e.Name)
		}
	}
}

func TestCLI_MaxDepth_Limits(t *testing.T) {
	bin := buildCLI(t)
	td := t.TempDir()
	_ = mk(t, td, "a.txt", 1)
	_ = mk(t, td, "d1/b.txt", 1)
	_ = mk(t, td, "d1/d2/c.txt", 1)

	run := func(depth int) []string {
		cmd := exec.Command(bin, "-root", td, "-json", "-max-depth", intToStr(depth), "-concurrency", "4")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = new(bytes.Buffer)
		if err := cmd.Run(); err != nil {
			t.Fatalf("run depth=%d: %v; stderr=%s", depth, err, cmd.Stderr.(*bytes.Buffer).String())
		}
		var arr []cliEntry
		if err := json.NewDecoder(&out).Decode(&arr); err != nil {
			t.Fatalf("decode: %v\nraw: %s", err, out.String())
		}
		var paths []string
		for _, e := range arr {
			if !e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		sort.Strings(paths)
		return paths
	}

	got0 := run(0)
	for _, p := range got0 {
		if filepath.Dir(p) != td {
			t.Fatalf("max-depth=0 should include only root-level; got %q", p)
		}
	}

	gotAll := run(-1)
	if len(gotAll) < len(got0) {
		t.Fatalf("max-depth=-1 should be superset")
	}

	_ = runtime.GOMAXPROCS(1)
}

func TestCLI_NDJSON(t *testing.T) {
	bin := buildCLI(t)
	td := t.TempDir()
	_ = mk(t, td, "a.txt", 1)

	cmd := exec.Command(bin, "-root", td, "-ndjson")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected some output")
	}
	for _, ln := range lines {
		var e cliEntry
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			t.Fatalf("invalid json line %q: %v", ln, err)
		}
	}
}

func TestCLI_PrettyJSON(t *testing.T) {
	bin := buildCLI(t)
	td := t.TempDir()
	_ = mk(t, td, "a.txt", 1)

	cmd := exec.Command(bin, "-root", td, "-json", "-pretty")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
	}
	if !strings.Contains(out.String(), "\n  ") {
		t.Fatalf("expected pretty JSON indentation, got: %s", out.String())
	}
	var arr []cliEntry
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, out.String())
	}
}

func TestCLI_OutFile(t *testing.T) {
	bin := buildCLI(t)
	td := t.TempDir()
	_ = mk(t, td, "a.txt", 1)
	outFile := filepath.Join(td, "out.json")

	cmd := exec.Command(bin, "-root", td, "-json", "-out", outFile)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output with -out; got %q", stdout.String())
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	var arr []cliEntry
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, string(data))
	}
	if len(arr) == 0 {
		t.Fatalf("expected some entries in output file")
	}
}

func TestCLI_FollowSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require special privileges on Windows")
	}
	bin := buildCLI(t)
	td := t.TempDir()
	realDir := filepath.Join(td, "real")
	_ = mk(t, realDir, "f.txt", 1)
	if err := os.Symlink(realDir, filepath.Join(td, "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	run := func(follow bool) []string {
		args := []string{"-root", td, "-json"}
		if follow {
			args = append(args, "-follow-symlinks")
		}
		cmd := exec.Command(bin, args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = new(bytes.Buffer)
		if err := cmd.Run(); err != nil {
			t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
		}
		var arr []cliEntry
		if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
			t.Fatalf("unmarshal: %v\nraw: %s", err, out.String())
		}
		var paths []string
		for _, e := range arr {
			if !e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		sort.Strings(paths)
		return paths
	}

	without := run(false)
	with := run(true)
	target := filepath.Join(td, "link", "f.txt")
	has := func(sl []string, s string) bool {
		for _, p := range sl {
			if p == s {
				return true
			}
		}
		return false
	}
	if has(without, target) {
		t.Fatalf("expected %s absent without follow-symlinks", target)
	}
	if !has(with, target) {
		t.Fatalf("expected %s present with follow-symlinks", target)
	}
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
