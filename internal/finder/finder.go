// Package finder provides the core file discovery and filtering engine used by gofind.
// Run walks the filesystem and writes matches. The branching handles many
// filter combinations. TODO(hamed): split into smaller helpers to reduce complexity.
//
//nolint:gocyclo // temporary; planned refactor will remove this
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
	"syscall"
	"time"
)

// OutputFormat controls how entries are written to the provided writer.
type OutputFormat int

const (
	// OutputText writes each matched path as a single line of plain text.
	OutputText OutputFormat = iota
	// OutputJSON writes a JSON array (streamed) of Entry values.
	OutputJSON
	// OutputNDJSON writes newline-delimited JSON entries.
	OutputNDJSON
)

// Config holds search options for the directory walk.
type Config struct {
	// Root is the starting directory.
	Root string
	// Extensions, when non-empty, includes only files with these lowercase extensions (e.g. ".go").
	Extensions map[string]bool
	// NameRegex, when set, must match the base name (file or directory) to be included.
	NameRegex *regexp.Regexp
	// MinSize and MaxSize constrain file sizes in bytes (0 = no bound). Directories are unaffected.
	MinSize int64
	MaxSize int64
	// After and Before filter by modification time (zero value = no bound).
	After  time.Time
	Before time.Time
	// IncludeHidden includes dotfiles on Unix (and simple Windows dotfile heuristic).
	IncludeHidden bool
	// MaxDepth controls recursion: -1 = unlimited, 0 = only children of root, 1 = one level deeper, etc.
	MaxDepth int
	// Concurrency is the max number of concurrent directory workers. <=0 defaults to NumCPU.
	Concurrency int
	// OutputFormat selects the output writer format.
	OutputFormat OutputFormat
	// PrettyJSON enables indentation for JSON/NDJSON outputs.
	PrettyJSON bool
	// FollowSymlinks descends into symlinked directories (with loop detection).
	FollowSymlinks bool
}

// Entry describes a matched filesystem entry (file or directory).
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

// Run executes the search using cfg, writing results to out.
// It streams output and returns when traversal completes or ctx is canceled.
func Run(ctx context.Context, out io.Writer, cfg Config) error {
	if err := cfg.validate(); err != nil {
		return err
	}

	// Track visited inodes (for follow-symlinks loop detection; best-effort on Unix).
	type inode struct {
		dev uint64
		ino uint64
	}
	inodeOf := func(fi fs.FileInfo) (inode, bool) {
		if st, ok := fi.Sys().(*syscall.Stat_t); ok {
			return inode{dev: uint64(st.Dev), ino: uint64(st.Ino)}, true
		}
		return inode{}, false
	}
	type inodeSet struct {
		mu sync.Mutex
		m  map[inode]struct{}
	}
	hasInode := func(s *inodeSet, i inode) bool {
		s.mu.Lock()
		_, ok := s.m[i]
		s.mu.Unlock()
		return ok
	}
	addInode := func(s *inodeSet, i inode) {
		s.mu.Lock()
		s.m[i] = struct{}{}
		s.mu.Unlock()
	}
	visited := &inodeSet{m: make(map[inode]struct{})}
	if cfg.FollowSymlinks {
		if rfi, err := os.Stat(cfg.Root); err == nil {
			if ino, ok := inodeOf(rfi); ok {
				addInode(visited, ino)
			}
		}
	}

	// Single writer goroutine to keep output safe and ordered.
	entryCh := make(chan Entry, 256)
	writeErr := make(chan error, 1)

	var wgWriter sync.WaitGroup
	wgWriter.Add(1)
	go func() {
		defer wgWriter.Done()
		var firstErr error
		record := func(err error) {
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		switch cfg.OutputFormat {
		case OutputJSON:
			if _, err := io.WriteString(out, "["); err != nil {
				record(err)
			}
			first := true
			for e := range entryCh {
				if firstErr != nil {
					// keep draining to avoid blocking producers
					continue
				}
				if !first {
					if cfg.PrettyJSON {
						_, _ = io.WriteString(out, ",\n")
					} else {
						_, _ = io.WriteString(out, ",")
					}
				} else if cfg.PrettyJSON {
					_, _ = io.WriteString(out, "\n")
				}
				first = false

				var b []byte
				var err error
				if cfg.PrettyJSON {
					b, err = json.MarshalIndent(e, "  ", "  ")
				} else {
					b, err = json.Marshal(e)
				}
				if err != nil {
					record(err)
					continue
				}
				if _, err := out.Write(b); err != nil {
					record(err)
					continue
				}
			}
			if firstErr == nil {
				if cfg.PrettyJSON {
					_, _ = io.WriteString(out, "\n")
				}
				_, _ = io.WriteString(out, "]")
			}
			if firstErr != nil {
				writeErr <- firstErr
			}
		case OutputNDJSON:
			enc := json.NewEncoder(out)
			enc.SetEscapeHTML(false)
			if cfg.PrettyJSON {
				enc.SetIndent("", "  ")
			}
			for e := range entryCh {
				if firstErr != nil {
					continue
				}
				if err := enc.Encode(e); err != nil {
					record(err)
					continue
				}
			}
			if firstErr != nil {
				writeErr <- firstErr
			}
		default:
			for e := range entryCh {
				if firstErr != nil {
					continue
				}
				if _, err := fmt.Fprintln(out, e.Path); err != nil {
					record(err)
					continue
				}
			}
			if firstErr != nil {
				writeErr <- firstErr
			}
		}
	}()

	// Bounded concurrency via semaphore.
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
			// Non-fatal: skip this subtree.
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
				continue
			}

			linfo, err := os.Lstat(full)
			if err != nil {
				continue
			}
			info := linfo
			isLink := linfo.Mode()&fs.ModeSymlink != 0
			if isLink && cfg.FollowSymlinks {
				if ti, err := os.Stat(full); err == nil {
					info = ti
				} else {
					continue
				}
			}
			isDir := info.IsDir()

			// Emit when filters match.
			if matches(&cfg, isDir, info) {
				entryCh <- Entry{
					Path:    full,
					Name:    name,
					Size:    info.Size(),
					Mode:    info.Mode(),
					ModTime: info.ModTime(),
					IsDir:   isDir,
				}
			}

			// Recurse into directories if within depth.
			if isDir {
				// Loop detection when following symlinks
				if cfg.FollowSymlinks {
					if ino, ok := inodeOf(info); ok {
						if hasInode(visited, ino) {
							continue
						}
						addInode(visited, ino)
					}
				}
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

func matches(cfg *Config, isDir bool, info fs.FileInfo) bool {
	name := info.Name()

	// extension filter (files only)
	if len(cfg.Extensions) > 0 && !isDir {
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
	if !isDir {
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

// stringsToLower is a tiny helper avoiding an extra strings import here.
func stringsToLower(s string) string {
	b := []rune(s)
	for i, r := range b {
		if 'A' <= r && r <= 'Z' {
			b[i] = r + ('a' - 'A')
		}
	}
	return string(b)
}
