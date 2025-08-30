package finder

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Hamed0406/gofind/internal/ignore"
)

type Config struct {
	Path             string
	RespectGitignore bool
	ExtraIgnores     []string
}

// Run walks from cfg.Path, applies ignore rules, and prints matching file paths.
// Returns number of printed entries.
func Run(ctx context.Context, out io.Writer, cfg Config) (int, error) {
	if cfg.Path == "" {
		cfg.Path = "."
	}
	m, err := ignore.New(ignore.Config{
		StartPath:        cfg.Path,
		RespectGitignore: cfg.RespectGitignore,
		ExtraPatterns:    cfg.ExtraIgnores,
	})
	if err != nil {
		return 0, err
	}

	var count int
	walkFn := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// If we can't read a directory, just continue searching.
			// Print the error to stderr but don't abort the whole walk.
			fmt.Fprintln(os.Stderr, walkErr)
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		isDir := d.IsDir()

		// Apply ignore rules
		if m.Enabled() && m.Match(path, isDir) {
			if isDir {
				return fs.SkipDir
			}
			return nil
		}

		// For MVP: print files only (skip directories)
		if !isDir {
			fmt.Fprintln(out, path)
			count++
		}
		return nil
	}

	if err := filepath.WalkDir(cfg.Path, walkFn); err != nil {
		// ctx.Err() means cancelled by user (Ctrl-C), don't treat as fatal
		if err == context.Canceled {
			return count, nil
		}
		return count, err
	}
	return count, nil
}
