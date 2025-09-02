//go:build windows

package finder

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func setHiddenAttr(t *testing.T, p string) {
	t.Helper()
	utf := syscall.StringToUTF16Ptr(p)
	const FILE_ATTRIBUTE_HIDDEN = 0x2
	attrs, err := syscall.GetFileAttributes(utf)
	if err != nil {
		t.Fatalf("get attrs: %v", err)
	}
	if err := syscall.SetFileAttributes(utf, attrs|FILE_ATTRIBUTE_HIDDEN); err != nil {
		t.Fatalf("set attrs: %v", err)
	}
}

func TestHiddenWindowsAttribute(t *testing.T) {
	td := t.TempDir()
	vis := filepath.Join(td, "visible.txt")
	hid := filepath.Join(td, "hidden.txt")
	if err := os.WriteFile(vis, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hid, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	setHiddenAttr(t, hid)

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
		var names []string
		for _, e := range arr {
			names = append(names, filepath.Base(e.Path))
		}
		return names
	}

	got := run(false)
	for _, n := range got {
		if n == "hidden.txt" {
			t.Fatalf("hidden file should be excluded when IncludeHidden=false; got %v", got)
		}
	}

	got = run(true)
	hasHidden := false
	for _, n := range got {
		if n == "hidden.txt" {
			hasHidden = true
		}
	}
	if !hasHidden {
		t.Fatalf("expected hidden.txt when IncludeHidden=true; got %v", got)
	}
}
