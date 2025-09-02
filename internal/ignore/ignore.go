// Package ignore implements a minimal .gitignore-style matcher used by gofind.
package ignore

import (
	"path/filepath"
	"strings"
)

// Matcher evaluates whether a path should be ignored according to simple patterns.
type Matcher struct {
	enabled  bool
	root     string
	patterns []string
}

// Config configures the Matcher.
type Config struct {
	// Root is the base directory where patterns are evaluated from.
	Root string
	// Patterns is a list of glob-like patterns to ignore (e.g., "node_modules/", "*.tmp").
	Patterns []string
	// Enabled toggles matching on or off.
	Enabled bool
}

// New creates a new Matcher with the provided config.
func New(cfg Config) (*Matcher, error) {
	m := &Matcher{
		enabled:  cfg.Enabled,
		root:     cfg.Root,
		patterns: append([]string(nil), cfg.Patterns...),
	}
	return m, nil
}

// Match reports whether the given path (relative or absolute) should be ignored.
// If isDir is true, directory-only patterns (ending with "/") can apply.
// Semantics:
//   - "node_modules/" matches the directory itself AND anything under it.
//   - "*.tmp" matches basenames by glob.
//   - Simple prefix matching for directory globs.
func (m *Matcher) Match(path string, isDir bool) bool {
	if !m.enabled {
		return false
	}
	// Make path relative to root if possible.
	if m.root != "" {
		if rel, err := filepath.Rel(m.root, path); err == nil {
			path = rel
		}
	}
	path = filepath.ToSlash(path)

	for _, p := range m.patterns {
		pp := strings.TrimSpace(p)
		if pp == "" {
			continue
		}
		dirOnly := strings.HasSuffix(pp, "/")
		ppNoSlash := strings.TrimSuffix(pp, "/")

		// If pattern is directory-only:
		// - match the directory itself (when isDir && base == ppNoSlash)
		// - match any descendant (prefix "ppNoSlash/")
		if dirOnly {
			if isDir && filepath.Base(path) == ppNoSlash {
				return true
			}
			if strings.HasPrefix(path, ppNoSlash+"/") {
				return true
			}
			// Also match the directory exact relative path.
			if path == ppNoSlash {
				return true
			}
			continue
		}

		// File/basename glob match (e.g., "*.tmp")
		if ok, _ := filepath.Match(ppNoSlash, filepath.Base(path)); ok {
			return true
		}

		// Fallback: prefix match for simple directory-like globs without trailing slash.
		if strings.HasPrefix(path, ppNoSlash+"/") {
			return true
		}
	}
	return false
}

// Enabled reports whether matching is active.
func (m *Matcher) Enabled() bool { return m.enabled }

// Root returns the root directory used for relative path evaluation.
func (m *Matcher) Root() string { return m.root }
