# Rewind

A lightweight terminal-based file version tracker for AI-assisted coding workflows.

Instead of using Git commits for every experiment, Rewind provides quick file-level snapshots — save, navigate, revert, and compare file versions instantly.

## Install

```bash
go install ./cmd/rewind/
```

Make sure `$(go env GOPATH)/bin` is in your `PATH`.

## Usage

### Track a file

```bash
rewind track main.go
```

### Save a snapshot

```bash
rewind save main.go "before AI refactor"
```

### View history

```bash
rewind history main.go
```

### Revert to a previous version

```bash
rewind revert main.go v1
```

### Compare current file against a version

```bash
rewind diff main.go v1
```

### Auto-snapshot on file changes

```bash
rewind watch main.go
rewind watch main.go --diff    # show diff preview on each save
```

## How it works

- Snapshots are stored globally in `~/.rewind/`
- Files are content-addressed using SHA256 hashes
- Storage uses reverse delta compression — the latest version is always stored in full, older versions are stored as patches
- Deduplication means identical file contents are never stored twice

## Architecture

```
CLI (cmd/rewind)
 |
 v
Command Layer (internal/cli)
 |
 v
Core Services
 ├── Snapshot Service (internal/snapshot)
 │     ├── Storage Engine (internal/storage)
 │     ├── Metadata Manager (internal/metadata)
 │     └── Diff Engine (internal/diff)
 └── File Watcher (internal/watcher)
 |
 v
Global Storage (~/.rewind)
 ├── objects/   (content-addressable file store)
 └── files/     (per-file version metadata)
```

## Project Structure

```
rewind/
├── cmd/rewind/main.go            # entry point
├── internal/
│   ├── cli/commands.go            # cobra CLI commands
│   ├── snapshot/snapshot.go       # track, save, history, revert, diff
│   ├── snapshot/utils.go          # helper utilities
│   ├── storage/objects.go         # content-addressable object store
│   ├── metadata/metadata.go       # version metadata (JSON)
│   ├── diff/diff.go               # patch, apply, compute (go-diff)
│   └── watcher/watcher.go         # fsnotify file watcher with debounce
├── pkg/hash/hash.go               # SHA256 hashing utility
├── go.mod
└── go.sum
```

## Dependencies

| Package | Purpose |
|---|---|
| [cobra](https://github.com/spf13/cobra) | CLI framework |
| [go-diff](https://github.com/sergi/go-diff) | Diff and patch engine |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [bubbletea](https://github.com/charmbracelet/bubbletea) | Terminal UI (future) |
| [fsnotify](https://github.com/fsnotify/fsnotify) | File system watcher |
