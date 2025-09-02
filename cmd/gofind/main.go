// cmd/gofind/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Hamed0406/gofind/internal/finder"
)

func main() {
	var (
		root        = flag.String("root", ".", "root directory to search")
		extsCSV     = flag.String("ext", "", "comma-separated list of file extensions to include (e.g. \".go,.md\")")
		nameReStr   = flag.String("name-regex", "", "regex to match file/dir names")
		minSizeStr  = flag.String("min-size", "", "minimum size to include (e.g. 10KB, 2MB, 1G)")
		maxSizeStr  = flag.String("max-size", "", "maximum size to include (e.g. 500KB, 10MB)")
		afterStr    = flag.String("after", "", "include entries modified after this time (YYYY-MM-DD or RFC3339)")
		beforeStr   = flag.String("before", "", "include entries modified before this time (YYYY-MM-DD or RFC3339)")
		includeHid  = flag.Bool("include-hidden", false, "include hidden files (Unix dotfiles and Windows hidden attribute)")
		maxDepth    = flag.Int("max-depth", -1, "maximum directory depth (-1 = unlimited, 0 = only root's direct children)")
		jsonOut     = flag.Bool("json", false, "stream JSON output instead of plain lines")
		concurrency = flag.Int("concurrency", runtime.NumCPU(), "number of concurrent directory workers")
	)
	flag.Parse()

	cfg := finder.Config{
		Root:          *root,
		IncludeHidden: *includeHid,
		MaxDepth:      *maxDepth,
		Concurrency:   *concurrency,
		OutputFormat:  finder.OutputText,
	}

	// extensions
	if s := strings.TrimSpace(*extsCSV); s != "" {
		cfg.Extensions = make(map[string]bool)
		for _, e := range strings.Split(s, ",") {
			e = strings.ToLower(strings.TrimSpace(e))
			if e == "" {
				continue
			}
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			cfg.Extensions[e] = true
		}
	}

	// name regex
	if rs := strings.TrimSpace(*nameReStr); rs != "" {
		re, err := regexp.Compile(rs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --name-regex: %v\n", err)
			os.Exit(2)
		}
		cfg.NameRegex = re
	}

	// size filters
	if *minSizeStr != "" {
		n, err := parseSize(*minSizeStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --min-size: %v\n", err)
			os.Exit(2)
		}
		cfg.MinSize = n
	}
	if *maxSizeStr != "" {
		n, err := parseSize(*maxSizeStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --max-size: %v\n", err)
			os.Exit(2)
		}
		cfg.MaxSize = n
	}

	// time filters
	if *afterStr != "" {
		t, err := parseTime(*afterStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --after: %v\n", err)
			os.Exit(2)
		}
		cfg.After = t
	}
	if *beforeStr != "" {
		t, err := parseTime(*beforeStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid --before: %v\n", err)
			os.Exit(2)
		}
		cfg.Before = t
	}

	if *jsonOut {
		cfg.OutputFormat = finder.OutputJSON
	}

	ctx := context.Background()
	if err := finder.Run(ctx, os.Stdout, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "KB"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "GB"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "K"):
		mult = 1 << 10
		s = strings.TrimSuffix(s, "K")
	case strings.HasSuffix(s, "M"):
		mult = 1 << 20
		s = strings.TrimSuffix(s, "M")
	case strings.HasSuffix(s, "G"):
		mult = 1 << 30
		s = strings.TrimSuffix(s, "G")
	}
	val := strings.TrimSpace(s)
	var n int64
	_, err := fmt.Sscan(val, &n)
	if err != nil {
		return 0, fmt.Errorf("could not parse number %q", s)
	}
	return n * mult, nil
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	// Try YYYY-MM-DD
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		t, err := time.Parse("2006-01-02", s)
		if err == nil {
			return t, nil
		}
	}
	// Try RFC3339
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	// Try a relaxed datetime (YYYY-MM-DD HH:MM)
	if t, err := time.Parse("2006-01-02 15:04", s); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("unsupported time format")
}
