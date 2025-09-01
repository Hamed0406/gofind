package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTestDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	fp := filepath.Join(tmp, "f.txt")
	if err := os.WriteFile(fp, []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	return tmp
}

func TestRunNDJSON(t *testing.T) {
	root := makeTestDir(t)
	var buf bytes.Buffer
	if err := runNDJSON(root, false, &buf); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 { // directory + file
		t.Fatalf("expected at least 2 NDJSON lines, got %d", len(lines))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil {
		t.Fatalf("invalid JSON line: %v", err)
	}
	if _, ok := m["path"]; !ok {
		t.Fatalf("expected 'path' field")
	}
}

func TestRunJSONArray(t *testing.T) {
	root := makeTestDir(t)
	var buf bytes.Buffer
	if err := runJSONArray(root, false, &buf, true); err != nil {
		t.Fatal(err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &arr); err != nil {
		t.Fatalf("invalid JSON array: %v", err)
	}
	if len(arr) < 2 {
		t.Errorf("expected at least 2 results, got %d", len(arr))
	}
}

func TestRunHuman(t *testing.T) {
	root := makeTestDir(t)
	var buf bytes.Buffer
	if err := runHuman(root, false, &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "f.txt") {
		t.Errorf("expected human output to contain f.txt, got %s", out)
	}
	if !strings.Contains(out, root) {
		t.Errorf("expected output to contain root path")
	}
}
