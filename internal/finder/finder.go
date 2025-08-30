package finder

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/Hamed0406/gofind/internal/ignore"
)

type Config struct {
	Path             string
	RespectGitignore bool
	ExtraIgnores     []string

	Exts   []string // normalized to lower with leading dot
	Name   string   // substring (case-insensitive on Windows)
	Regex  string   // filename regex
	Type   string   // f|d|a
	Hidden bool     // include dotfiles
}

// Run walks and prints entries passing filters; returns count.
func Run(ctx context.Context, out io.Writer, cfg Config) (int, error) {
	if cfg.Path == "" {
		cfg.Path = "."
	}
	// normalize filters
	exts := normalizeExts(cfg.Exts)
	namePat := cfg.Name
	var re *regexp.Regexp
	if cfg.Regex != "" {
		rx, err := regexp.Compile(cfg.Regex)
		if err != nil {
			return 0, fmt.Errorf("invalid --regex: %w", err)
		}
		re = rx
	}
	typ := strings.ToLower(strings.TrimSpace(cfg.Type))
	if typ == "" {
		typ = "f"
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
			// log and continue
			fmt.Fprintln(os.Stderr, walkErr)
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		isDir := d.IsDir()

		// .gitignore and extra ignores
		if m.Enabled() && m.Match(path, isDir) {
			if isDir {
				return fs.SkipDir
			}
			return nil
		}

		// HIDDEN FILTER FIRST (so we can skip hidden directories)
		if !cfg.Hidden && isHidden(path, d) {
			if isDir {
				return fs.SkipDir
			}
			return nil
		}

		// type filter
		if typ == "f" && isDir {
			return nil
		}
		if typ == "d" && !isDir {
			return nil
		}

		// filename-based filters
		name := d.Name()

		// extension filter (files only)
		if !isDir && len(exts) > 0 && !matchExt(name, exts) {
			return nil
		}

		// name substring
		if namePat != "" && !matchName(name, namePat) {
			return nil
		}

		// regex on name
		if re != nil && !re.MatchString(name) {
			return nil
		}

		// print match
		fmt.Fprintln(out, path)
		count++
		return nil
	}

	if err := filepath.WalkDir(cfg.Path, walkFn); err != nil {
		if err == context.Canceled {
			return count, nil
		}
		return count, err
	}
	return count, nil
}

// --- helpers ---

func normalizeExts(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		out = append(out, strings.ToLower(e))
	}
	return out
}

func matchExt(name string, exts []string) bool {
	low := strings.ToLower(name)
	for _, e := range exts {
		if strings.HasSuffix(low, e) {
			return true
		}
	}
	return false
}

func matchName(name, needle string) bool {
	if runtime.GOOS == "windows" {
		return strings.Contains(strings.ToLower(name), strings.ToLower(needle))
	}
	return strings.Contains(name, needle)
}

// MVP hidden: dotfiles/dirs (Unix). Windows attribute hidden comes later.
func isHidden(_ string, d fs.DirEntry) bool {
	n := d.Name()
	if n == "." || n == ".." {
		return false
	}
	return strings.HasPrefix(n, ".")
}
