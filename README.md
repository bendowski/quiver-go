# Quiver

Quiver turns Go source code into a property graph stored in [LadyBugDB](https://github.com/LadybugDB/go-ladybug), then lets you query that graph with Cypher or reconstruct source files from it.

## What It Does

- Parses `.go` files and extracts code structure into graph nodes and edges.
- Stores that graph in LadyBugDB (in-memory by default, or on disk with `--db`).
- Executes raw Cypher queries against the stored graph.
- Dumps source files back to disk from stored `File.source` content, with a fallback reconstruction path.

## CLI Overview

The CLI entrypoint is `cmd/quiver` and exposes three commands:

- `quiver load <dir>`
- `quiver query <cypher>`
- `quiver dump <outdir>`

Global flag:

- `--db <path>`: LadyBug database directory. If omitted, Quiver uses an in-memory DB for that process run.

### `load`

`load` parses Go files in the provided directory and inserts nodes/edges into LadyBug.

Important behavior:

- `ParseDir` scans `.go` files **non-recursively** in `<dir>`.
- Files that fail to parse are skipped.
- Node and edge insert errors are counted and reported; command continues loading remaining items.

### `query`

`query` executes a raw Cypher string and prints tabular output:

- Header row with column names.
- All rows returned by LadyBug.
- Final row count summary.

### `dump`

`dump` writes all `File` nodes to `<outdir>`.

- If a file node has `source`, that content is written directly.
- If `source` is empty, Quiver reconstructs file content from contained nodes (`Import`, `TypeDecl`, `Function`, `Variable`) and tries to format it with `go/format`.

## Graph Schema

### Node kinds

- `Package`
- `File`
- `Import`
- `Function`
- `TypeDecl`
- `Variable`
- `Field`

### Edge kinds

- `CONTAINS`
- `IMPORTS`
- `HAS_RECEIVER`
- `EMBEDS`
- `CALLS`
- `REFERS_TO`
- `IMPLEMENTS`
- `RESOLVES_TO`

Note: current `load` flow (via `ParseDir`) produces structural edges (`CONTAINS`, `IMPORTS`, `HAS_RECEIVER`, `EMBEDS`). Other relationship tables exist in schema and can support richer analysis when type-aware parsing (`ParsePackages`) data is used.

## Parse Output Details

For parsed declarations, Quiver stores metadata such as:

- name/signature/kind
- source snippet
- file path
- start/end lines
- exported flag
- doc comment text (where applicable)

This makes Cypher queries useful for code indexing tasks like:

- listing exported APIs
- finding methods by receiver type
- locating embedded struct relationships
- searching declarations by path/line metadata

## Build

Quiver depends on [go-ladybug](https://github.com/LadybugDB/go-ladybug), which wraps a native LadyBugDB library via cgo. This repo uses go-ladybug's "Option 2" setup (see its [README](https://github.com/LadybugDB/go-ladybug#option-2-add-the-compiled-libraries-to-your-project)): the native shared library is downloaded on demand into `cmd/quiver/lib-ladybug/` (gitignored) instead of being vendored into the module cache.

1.  Download the native library (one-time, or after bumping the go-ladybug version):

    ```bash
    go generate -tags system_ladybug ./cmd/quiver/...
    ```

    **Known quirk:** the upstream generate script can produce a circular symlink (`liblbug.dylib` ↔ `liblbug.0.dylib` on macOS) instead of pointing at the real versioned file it just downloaded (e.g. `liblbug.0.18.2.dylib`). If the build below fails with `library 'lbug' not found`, check `cmd/quiver/lib-ladybug/` and fix the symlinks:

    ```bash
    ln -sf liblbug.<version>.dylib cmd/quiver/lib-ladybug/liblbug.dylib
    ln -sf liblbug.<version>.dylib cmd/quiver/lib-ladybug/liblbug.0.dylib
    ```

2.  Build with the `system_ladybug` tag:

    ```bash
    go build -tags system_ladybug -o quiver ./cmd/quiver
    ```

## Quick Start

```bash
# load Go files from a directory into a persistent DB
./quiver --db /tmp/quiver.db load ./goparse

# query the graph
./quiver --db /tmp/quiver.db query "MATCH (f:Function) RETURN f.name, f.file_path ORDER BY f.name"

# reconstruct files into an output directory
./quiver --db /tmp/quiver.db dump /tmp/quiver-out
```

## Package Map

- `cmd/quiver`: CLI commands and wiring.
- `goparse`: AST parsing and graph extraction.
- `model`: node/edge types and enums.
- `schema`: LadyBug node/relationship table DDL.
- `store`: storage interface.
- `store/ladybug`: LadyBug-backed `store.Store` implementation.
- `query/cypher`: Cypher query builder + query executor wrapper.
- `dump`: file export and fallback reconstruction logic.

## Development

Run tests (other packages besides `cmd/quiver` import go-ladybug directly and build their own test binaries, so `CGO_CFLAGS`/`CGO_LDFLAGS` need to be set explicitly rather than relying on the cgo pragma in `cmd/quiver/ladybug_lib.go`):

```bash
CGO_CFLAGS="-I$(pwd)/cmd/quiver/lib-ladybug" \
CGO_LDFLAGS="-L$(pwd)/cmd/quiver/lib-ladybug -Wl,-rpath,$(pwd)/cmd/quiver/lib-ladybug" \
go test -tags system_ladybug ./...
```

## Upgrading Dependencies

```bash
go get -u ./...
go mod tidy
```

If this bumps `github.com/LadybugDB/go-ladybug`, re-run the native library download (`go generate -tags system_ladybug ./cmd/quiver/...`, see [Build](#build)) and re-verify the build/tests, since go-ladybug has changed its native-loading strategy across major-ish versions before (e.g. the v0.13.1 → v0.17.0 jump moved from a module-cache-vendored dylib to the `system_ladybug` tag setup described above).
