# Gofind

Gofind is a fast cross-platform file finder written in Go. It walks a directory tree and prints entries that match user-defined filters while respecting your project's `.gitignore` file.

## Features

- Respect `.gitignore` files and custom ignore patterns.
- Filter by file extension, filename substring, or regular expression.
- Limit results to files, directories, or both.
- Optionally include hidden files.
- Works on all major platforms.

## Installation

```bash
go install github.com/Hamed0406/gofind/cmd/gofind@latest
```

Or build from source:

```bash
make build
```

## Usage

```bash
gofind [flags]
```

Key flags:

- `--root` — root directory to scan (default ".").
- `--json` — emit results as a JSON array.
- `--ndjson` — emit newline-delimited JSON.
- `--pretty` — pretty-print JSON (with `--json` or `--ndjson`).
- `--out` — write output to a file instead of stdout.
- `--follow-symlinks` — resolve symlinks and include targets.
- `--version` — print version and exit.
- `--ext` — comma-separated list of file extensions to include (e.g. ".go,.md").
- `--name-regex` — regular expression to match file or directory names.
- `--min-size` / `--max-size` — include entries within a size range (e.g. "10KB", "2MB").
- `--after` / `--before` — filter by modification time (YYYY-MM-DD or RFC3339).
- `--include-hidden` — include hidden files and directories.
- `--max-depth` — limit directory traversal depth (-1 for unlimited).
- `--concurrency` — number of concurrent directory workers.

Example:

```bash
gofind --ext .go,.md
gofind --name-regex 'test.*\\.go'
gofind --min-size 10KB --max-size 2MB
gofind --after 2024-01-01 --before 2024-02-01
gofind --include-hidden
gofind --max-depth 2
gofind --concurrency 4
gofind --root . --json --pretty
```

## JSON / NDJSON output

The default output is human-readable. For automation or scripting, use JSON:

```bash
# Pretty JSON array to stdout
gofind --root . --json --pretty

# Streaming NDJSON to a file (best for huge trees)
gofind --root . --ndjson --out results.ndjson

# Follow symlinks and include their targets
gofind --root /opt --ndjson --follow-symlinks

```
## Testing

```bash
make test
```

## License

MIT
