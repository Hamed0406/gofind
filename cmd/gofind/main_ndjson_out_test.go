package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_NDJSON_Output(t *testing.T) {
	bin := buildCLI(t)
	td := t.TempDir()
	if err := os.WriteFile(filepath.Join(td, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "-root", td, "-ndjson")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) == 0 {
		t.Fatalf("expected some NDJSON output")
	}
	for _, ln := range lines {
		var e cliEntry
		if err := json.Unmarshal([]byte(ln), &e); err != nil {
			t.Fatalf("invalid NDJSON line %q: %v", ln, err)
		}
	}
	// sanity: NDJSON should NOT start with '['
	if strings.HasPrefix(strings.TrimSpace(out.String()), "[") {
		t.Fatalf("NDJSON should not be a JSON array")
	}
}

func TestCLI_PrettyJSON_Array(t *testing.T) {
	bin := buildCLI(t)
	td := t.TempDir()
	if err := os.WriteFile(filepath.Join(td, "b.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "-root", td, "-json", "-pretty")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
	}
	if !strings.Contains(out.String(), "\n  ") {
		t.Fatalf("expected pretty indentation; got: %s", out.String())
	}
	var arr []cliEntry
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, out.String())
	}
	if len(arr) == 0 {
		t.Fatalf("expected at least one entry")
	}
}

func TestCLI_OutFile_WritesAndMutesStdout(t *testing.T) {
	bin := buildCLI(t)
	td := t.TempDir()
	if err := os.WriteFile(filepath.Join(td, "c.txt"), []byte("z"), 0o644); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(td, "out.json")

	cmd := exec.Command(bin, "-root", td, "-json", "-out", outFile)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = new(bytes.Buffer)
	if err := cmd.Run(); err != nil {
		t.Fatalf("run: %v; stderr=%s", err, cmd.Stderr.(*bytes.Buffer).String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout with -out; got %q", stdout.String())
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
