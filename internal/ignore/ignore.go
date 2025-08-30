package ignore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
)

type Matcher struct {
	root    string
	ig      *gitignore.GitIgnore
	enabled bool
}

type Config struct {
	StartPath        string
	RespectGitignore bool
	ExtraPatterns    []string
}

func New(cfg Config) (*Matcher, error) {
	if cfg.StartPath == "" {
		cfg.StartPath = "."
	}
	root, err := findRoot(cfg.StartPath)
	if err != nil {
		return nil, err
	}

	var lines []string

	if cfg.RespectGitignore {
		p := filepath.Join(root, ".gitignore")
		if data, err := os.ReadFile(p); err == nil {
			text := strings.ReplaceAll(string(data), "\r\n", "\n")
			for _, ln := range strings.Split(text, "\n") {
				ln = strings.TrimSpace(ln)
				if ln == "" || strings.HasPrefix(ln, "#") {
					continue
				}
				lines = append(lines, ln)
			}
		}
	}

	for _, pat := range cfg.ExtraPatterns {
		if s := strings.TrimSpace(pat); s != "" {
			lines = append(lines, s)
		}
	}

	ig := gitignore.CompileIgnoreLines(lines...)
	return &Matcher{
		root:    root,
		ig:      ig,
		enabled: cfg.RespectGitignore || len(cfg.ExtraPatterns) > 0,
	}, nil
}

func (m *Matcher) Match(path string, isDir bool) bool {
	if !m.enabled {
		return false
	}
	rel := path
	if filepath.IsAbs(rel) {
		if r, err := filepath.Rel(m.root, rel); err == nil {
			rel = r
		}
	}
	rel = toSlash(rel)
	if isDir && !strings.HasSuffix(rel, "/") {
		rel += "/"
	}
	return m.ig.MatchesPath(rel)
}

func (m *Matcher) Enabled() bool { return m.enabled }
func (m *Matcher) Root() string  { return m.root }

// --- helpers ---

func findRoot(start string) (string, error) {
	start = filepath.Clean(start)
	info, err := os.Stat(start)
	if err != nil {
		return "", err
	}
	cur := start
	if !info.IsDir() {
		cur = filepath.Dir(start)
	}
	for {
		if hasGitDir(cur) {
			return absDir(cur)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return absDir(start)
}

func hasGitDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

func absDir(p string) (string, error) {
	ap, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(ap)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", errors.New("path is not a directory: " + ap)
	}
	return ap, nil
}

func toSlash(p string) string {
	p = filepath.Clean(p)
	p = strings.TrimPrefix(p, string(filepath.Separator))
	return filepath.ToSlash(p)
}
