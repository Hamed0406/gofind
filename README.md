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
- `--pretty` — pretty-print JSON (with `--json`).
- `--out` — write output to a file instead of stdout.
- `--follow-symlinks` — resolve symlinks and include targets.
- `--version` — print version and exit.

Example:

```bash
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
