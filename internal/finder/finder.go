package finder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

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

	Larger  string // e.g. "100M"
	Smaller string // e.g. "1G"
	Since   string // e.g. "7d" or "2025-08-01"
	Output  string // "path" | "json" | "ndjson"
}

// Result is an entry that matched filters.
type Result struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_dir"`
}

// Run walks and prints entries passing filters; returns count.
// nolint:gocyclo // TODO: break Run into smaller helpers (filters, walk, print)
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

	var largerBytes *int64
	if cfg.Larger != "" {
		v, err := parseSize(cfg.Larger)
		if err != nil {
			return 0, fmt.Errorf("invalid --larger: %w", err)
		}
		largerBytes = &v
	}

	var smallerBytes *int64
	if cfg.Smaller != "" {
		v, err := parseSize(cfg.Smaller)
		if err != nil {
			return 0, fmt.Errorf("invalid --smaller: %w", err)
		}
		smallerBytes = &v
	}

	var sinceTime *time.Time
	if cfg.Since != "" {
		t, err := parseSince(cfg.Since)
		if err != nil {
			return 0, fmt.Errorf("invalid --since: %w", err)
		}
		sinceTime = &t
	}

	// small tolerance so boundary mtimes are included
	var sinceCutoff *time.Time
	if sinceTime != nil {
		cut := sinceTime.Add(-2 * time.Second) // epsilon
		sinceCutoff = &cut
	}

	outMode := strings.ToLower(strings.TrimSpace(cfg.Output))
	if outMode == "" {
		outMode = "path"
	}
	if outMode != "path" && outMode != "json" && outMode != "ndjson" {
		return 0, errors.New("--output must be path|json|ndjson")
	}

	m, err := ignore.New(ignore.Config{
		StartPath:        cfg.Path,
		RespectGitignore: cfg.RespectGitignore,
		ExtraPatterns:    cfg.ExtraIgnores,
	})
	if err != nil {
		return 0, err
	}

	var (
		count   int
		enc     *json.Encoder
		results []Result
	)
	if outMode == "ndjson" {
		enc = json.NewEncoder(out)
	}

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

		// name-based filters
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

		// size/time filters need FileInfo (stat)
		info, err := d.Info()
		if err != nil {
			// if we can't stat, skip
			return nil
		}
		size := info.Size()
		mod := info.ModTime()

		// size filters
		if largerBytes != nil && !(size > *largerBytes) {
			return nil
		}
		if smallerBytes != nil && !(size < *smallerBytes) {
			return nil
		}

		// since filter (modtime >= threshold, with small tolerance)
		if sinceCutoff != nil && mod.Before(*sinceCutoff) {
			return nil
		}

		// Emit
		res := Result{Path: path, Size: size, ModTime: mod, IsDir: isDir}
		switch outMode {
		case "path":
			if _, werr := fmt.Fprintln(out, path); werr != nil {
				// best-effort write; ignore error
				_ = werr
			}
		case "ndjson":
			_ = enc.Encode(res)
		case "json":
			results = append(results, res)
		}
		count++
		return nil
	}

	if err := filepath.WalkDir(cfg.Path, walkFn); err != nil {
		if err == context.Canceled {
			// user canceled; still return what we printed
			return count, nil
		}
		return count, err
	}

	if outMode == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return count, err
		}
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

// parseSize supports "123", "10K", "100M", "2G" (base 1024).
func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, errors.New("empty size")
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "K"):
		mult = 1024
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "M"):
		mult = 1024 * 1024
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "G"):
		mult = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "G")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0, err
	}
	return n * mult, nil
}

// parseSince: "7d", "3h", "15m", "45s" or a date "YYYY-MM-DD" or RFC3339.
func parseSince(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty since")
	}

	// duration forms
	last := s[len(s)-1]
	if last == 'd' || last == 'h' || last == 'm' || last == 's' {
		unit := last
		num := strings.TrimSpace(s[:len(s)-1])
		val, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return time.Time{}, err
		}
		var dur time.Duration
		switch unit {
		case 'd':
			dur = time.Duration(val*24) * time.Hour
		case 'h':
			dur = time.Duration(val) * time.Hour
		case 'm':
			dur = time.Duration(val) * time.Minute
		case 's':
			dur = time.Duration(val) * time.Second
		}
		return time.Now().Add(-dur), nil
	}

	// date form
	if len(s) == 10 { // YYYY-MM-DD
		if t, err := time.Parse("2006-01-02", s); err == nil {
			return t, nil
		}
	}

	// RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unrecognized since format: %q", s)
}
