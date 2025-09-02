package finder

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type failWriter struct {
	failAfter int // number of successful writes allowed before failing
	writes    int
}

func (f *failWriter) Write(p []byte) (int, error) {
	if f.writes >= f.failAfter {
		return 0, errors.New("simulated writer failure")
	}
	f.writes++
	// pretend we wrote everything
	return len(p), nil
}

func makeTree(t *testing.T) string {
	t.Helper()
	td := t.TempDir()
	_ = os.WriteFile(filepath.Join(td, "a.txt"), []byte("x"), 0o644)
	_ = os.Mkdir(filepath.Join(td, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(td, "sub", "b.txt"), []byte("y"), 0o644)
	return td
}

func TestWriterFailure_Text_NoDeadlockAndError(t *testing.T) {
	td := makeTree(t)
	cfg := Config{
		Root:         td,
		Concurrency:  runtime.GOMAXPROCS(0),
		OutputFormat: OutputText,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	fw := &failWriter{failAfter: 0} // fail on first write
	err := Run(ctx, fw, cfg)
	if err == nil {
		t.Fatalf("expected error from writer failure")
	}
}

func TestWriterFailure_JSON_NoDeadlockAndError(t *testing.T) {
	td := makeTree(t)
	cfg := Config{
		Root:         td,
		Concurrency:  runtime.GOMAXPROCS(0),
		OutputFormat: OutputJSON,
		PrettyJSON:   true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// allow the initial "[" to succeed, then fail
	fw := &failWriter{failAfter: 1}
	err := Run(ctx, fw, cfg)
	if err == nil {
		t.Fatalf("expected error from writer failure")
	}
}
