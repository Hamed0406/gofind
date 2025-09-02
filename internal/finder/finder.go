// internal/finder/finder.go
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
	"sync"
	"time"
)

type OutputFormat int

const (
	OutputText OutputFormat = iota
	OutputJSON
)

type Config struct {
	Root          string
	Extensions    map[string]bool // include only these extensions (empty = all)
	NameRegex     *regexp.Regexp  // optional name filter
	MinSize       int64           // bytes, 0 = no min
	MaxSize       int64           // bytes, 0 = no max
	After         time.Time       // zero = no lower bound
	Before        time.Time       // zero = no upper bound
	IncludeHidden bool
	MaxDepth      int // -1 unlimited, 0 = only direct children, etc.
	Concurrency   int // default: NumCPU()
	OutputFormat  OutputFormat
}

type Entry struct {
	Path    string      `json:"path"`
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Mode    fs.FileMode `json:"mode"`
	ModTime time.Time   `json:"modTime"`
	IsDir   bool        `json:"isDir"`
}

func (c *Config) validate() error {
	if c.Root == "" {
		return errors.New("root directory is required")
	}
	if c.Concurrency <= 0 {
		c.Concurrency = runtime.NumCPU()
	}
	return nil
}

func Run(ctx context.Context, out io.Writer, cfg Config) error {
	if err := cfg.validate(); err != nil {
		return err
	}

	// Writer goroutine (single writer to keep output ordered and safe)
	entryCh := make(chan Entry, 256)
	writeErr := make(chan error, 1)
	var wgWriter sync.WaitGroup
	wgWriter.Add(1)
	go func() {
		defer wgWriter.Done()
		switch cfg.OutputFormat {
		case OutputJSON:
			// Stream a JSON array
			if _, err := io.WriteString(out, "["); err != nil {
				writeErr <- err
				return
			}
			first := true
			enc := json.NewEncoder(out)
			for e := range entryCh {
				if !first {
					_, _ = io.WriteString(out, ",")
				}
				first = false
				// enc.Encode adds a newline; that's fine for streaming
				if err := enc.Encode(e); err != nil {
					writeErr <- err
					return
				}
			}
			_, _ = io.WriteString(out, "]")
		default:
			for e := range entryCh {
				if _, werr := fmt.Fprintln(out, e.Path); werr != nil {
					// best-effort write; ignore error
					_ = werr
				}
			}
		}
	}()

	// Walk with bounded concurrency via a semaphore.
	type walkReq struct {
		path  string
		depth int
	}
	sem := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup

	var walk func(string, int)
	walk = func(dir string, depth int) {
		defer wg.Done()
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-sem }()

		entries, err := os.ReadDir(dir)
		if err != nil {
			// Non-fatal; just skip this subtree.
			return
		}
		for _, de := range entries {
			select {
			case <-ctx.Done():
				return
			default:
			}
			name := de.Name()
			full := filepath.Join(dir, name)

			// Hidden?
			if !cfg.IncludeHidden && isHidden(full, name) {
				// Skip hidden entries entirely
				continue
			}

			info, err := de.Info()
			if err != nil {
				continue
			}

			// Emit entry if it matches filters.
			if matches(&cfg, de, info) {
				entryCh <- Entry{
					Path:    full,
					Name:    name,
					Size:    info.Size(),
					Mode:    info.Mode(),
					ModTime: info.ModTime(),
					IsDir:   de.IsDir(),
				}
			}

			// Recurse into directories if within depth.
			if de.IsDir() {
				if cfg.MaxDepth >= 0 && depth >= cfg.MaxDepth {
					continue
				}
				wg.Add(1)
				go walk(full, depth+1)
			}
		}
	}

	// Kick off
	wg.Add(1)
	go walk(cfg.Root, 0)
	wg.Wait()
	close(entryCh)
	wgWriter.Wait()

	select {
	case err := <-writeErr:
		return err
	default:
		return nil
	}
}

func matches(cfg *Config, de fs.DirEntry, info fs.FileInfo) bool {
	name := info.Name()

	// extension filter (files only)
	if len(cfg.Extensions) > 0 && !de.IsDir() {
		ext := stringsToLower(filepath.Ext(name))
		if !cfg.Extensions[ext] {
			return false
		}
	}

	// name regex
	if cfg.NameRegex != nil && !cfg.NameRegex.MatchString(name) {
		return false
	}

	// size (files only)
	if !de.IsDir() {
		if cfg.MinSize > 0 && info.Size() < cfg.MinSize {
			return false
		}
		if cfg.MaxSize > 0 && info.Size() > cfg.MaxSize {
			return false
		}
	}

	// mod time
	if !cfg.After.IsZero() && info.ModTime().Before(cfg.After) {
		return false
	}
	if !cfg.Before.IsZero() && info.ModTime().After(cfg.Before) {
		return false
	}

	return true
}

// tiny helper (avoids importing strings everywhere in this file)
func stringsToLower(s string) string {
	b := []rune(s)
	for i, r := range b {
		if 'A' <= r && r <= 'Z' {
			b[i] = r + ('a' - 'A')
		}
	}
	return string(b)
}
