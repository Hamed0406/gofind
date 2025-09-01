package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Row represents one file or directory entry in the output.
type Row struct {
	Path      string      `json:"path"`
	Name      string      `json:"name"`
	Size      int64       `json:"size"`
	Mode      os.FileMode `json:"mode"`
	IsDir     bool        `json:"isDir"`
	ModTime   time.Time   `json:"modTime"`
	Ext       string      `json:"ext"`
	Root      string      `json:"root"`
	RelPath   string      `json:"relPath"`
	SymlinkTo string      `json:"symlinkTo,omitempty"`
}

func main() {
	var (
		root     string
		jsonOut  bool
		ndjson   bool
		pretty   bool
		outFile  string
		followSL bool
	)

	flag.StringVar(&root, "root", ".", "root directory to scan")
	flag.BoolVar(&jsonOut, "json", false, "emit JSON array to stdout or --out")
	flag.BoolVar(&ndjson, "ndjson", false, "emit Newline-Delimited JSON (streaming)")
	flag.BoolVar(&pretty, "pretty", false, "pretty-print JSON (with --json)")
	flag.StringVar(&outFile, "out", "", "write output to file instead of stdout")
	flag.BoolVar(&followSL, "follow-symlinks", false, "resolve symlinks (store target in symlinkTo)")
	flag.Parse()

	// Open output
	var out *os.File
	var err error
	if outFile != "" {
		out, err = os.Create(outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open output file: %v\n", err)
			os.Exit(1)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	switch {
	case ndjson:
		if err := runNDJSON(root, followSL, out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case jsonOut:
		if err := runJSONArray(root, followSL, out, pretty); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		if err := runHuman(root, followSL, out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func buildRow(root, path string, d fs.DirEntry, followSL bool) (Row, error) {
	fi, err := d.Info()
	if err != nil {
		return Row{}, err
	}

	rel, _ := filepath.Rel(root, path)
	ext := ""
	if !fi.IsDir() {
		ext = filepath.Ext(fi.Name())
	}

	r := Row{
		Path:    path,
		Name:    fi.Name(),
		Size:    fi.Size(),
		Mode:    fi.Mode(),
		IsDir:   fi.IsDir(),
		ModTime: fi.ModTime().UTC(),
		Ext:     ext,
		Root:    root,
		RelPath: rel,
	}

	if followSL && d.Type()&os.ModeSymlink != 0 {
		if target, e := os.Readlink(path); e == nil {
			r.SymlinkTo = target
		}
	}
	return r, nil
}

func walk(root string, followSL bool, onRow func(Row) error) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("root is not a directory: %s", root)
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries but continue walking
			return nil
		}
		row, e := buildRow(root, path, d, followSL)
		if e != nil {
			return nil
		}
		return onRow(row)
	})
}

func runNDJSON(root string, followSL bool, out io.Writer) error {
	bw := bufio.NewWriter(out)
	defer bw.Flush()
	enc := json.NewEncoder(bw)
	return walk(root, followSL, func(r Row) error {
		return enc.Encode(r) // one JSON object per line
	})
}

func runJSONArray(root string, followSL bool, out io.Writer, pretty bool) error {
	rows := make([]Row, 0, 1024)
	if err := walk(root, followSL, func(r Row) error {
		rows = append(rows, r)
		return nil
	}); err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	if pretty {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(rows)
}

func runHuman(root string, followSL bool, out io.Writer) error {
	bw := bufio.NewWriter(out)
	defer bw.Flush()
	return walk(root, followSL, func(r Row) error {
		_, _ = fmt.Fprintf(bw, "%12d  %s  %s\n",
			r.Size, r.ModTime.Format("2006-01-02 15:04:05"), r.Path)
		return nil
	})
}
