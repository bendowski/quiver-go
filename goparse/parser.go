// Package goparse parses Go source code into model.Node and model.Edge values.
package goparse

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/tools/go/packages"

	"github.com/bendowski/quiver/model"
	"github.com/google/uuid"
)

// ParseResult holds all nodes and edges produced by a parse run.
type ParseResult struct {
	Nodes []model.Node
	Edges []model.Edge

	// LoadErrors holds non-fatal per-package errors reported by
	// packages.Load during ParsePackages (parse or type-check problems).
	// The graph still contains whatever was parseable. Always nil for
	// ParseDir.
	LoadErrors []error
}

// Parser parses Go source trees into property graph elements.
type Parser struct {
	fs afero.Fs
}

// New returns a Parser that reads source files from fs.
func New(fs afero.Fs) *Parser {
	return &Parser{fs: fs}
}

// ParseDir parses all .go files in dir (non-recursively) using the provided
// Afero filesystem. It produces Package, File, Import, Function, TypeDecl,
// Variable, and Field nodes as well as CONTAINS, IMPORTS, HAS_RECEIVER, and
// EMBEDS edges.
//
// ParseDir does not perform type-checking; it relies solely on go/parser.
func (p *Parser) ParseDir(_ context.Context, dir string) (*ParseResult, error) {
	fset := token.NewFileSet()

	entries, err := afero.ReadDir(p.fs, dir)
	if err != nil {
		return nil, fmt.Errorf("ParseDir ReadDir %s: %w", dir, err)
	}

	// Read and parse every .go file once; files with syntax errors are
	// skipped.
	type parsedFile struct {
		name string
		path string
		src  []byte
		ast  *ast.File
	}
	var files []parsedFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		filePath := filepath.Join(dir, e.Name())
		src, err := afero.ReadFile(p.fs, filePath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filePath, err)
		}
		astFile, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
		if err != nil {
			continue
		}
		files = append(files, parsedFile{name: e.Name(), path: filePath, src: src, ast: astFile})
	}
	if len(files) == 0 {
		return &ParseResult{}, nil
	}

	// The Package node takes its name from the first parsed file; each File
	// node records its own package clause, since a directory can mix
	// package x and x_test files.
	pkgNode := model.Node{
		ID:   uuid.NewString(),
		Kind: model.KindPackage,
		Properties: map[string]any{
			model.PropName:       files[0].ast.Name.Name,
			model.PropImportPath: dir,
			model.PropDir:        dir,
		},
	}

	state := &parseState{
		typesByName: make(map[string]string),
	}
	state.nodes = append(state.nodes, pkgNode)

	for _, f := range files {
		fileID := uuid.NewString()
		fileNode := model.Node{
			ID:   fileID,
			Kind: model.KindFile,
			Properties: map[string]any{
				model.PropName:        f.name,
				model.PropFilePath:    f.path,
				model.PropPackageName: f.ast.Name.Name,
				model.PropSource:      string(f.src),
			},
		}
		state.nodes = append(state.nodes, fileNode)

		// CONTAINS: Package → File
		state.edges = append(state.edges, model.Edge{
			Kind:       model.EdgeContains,
			SourceID:   pkgNode.ID,
			SourceKind: model.KindPackage,
			TargetID:   fileID,
			TargetKind: model.KindFile,
		})

		v := &visitor{
			fset:     fset,
			src:      f.src,
			fileID:   fileID,
			filePath: f.path,
			state:    state,
		}
		for _, decl := range f.ast.Decls {
			v.visitDecl(decl)
		}
	}

	state.resolveDeferred()

	return &ParseResult{Nodes: state.nodes, Edges: state.edges}, nil
}

// ParsePackages uses golang.org/x/tools/go/packages to load and type-check
// the given patterns. It currently produces the same structural nodes and
// edges as ParseDir; the loaded type information is not used yet.
//
// TODO(#1): use type info to resolve CALLS, IMPLEMENTS, and REFERS_TO edges.
//
// Unlike ParseDir this requires a real filesystem and proper module setup.
func (p *Parser) ParsePackages(ctx context.Context, patterns ...string) (*ParseResult, error) {
	cfg := &packages.Config{
		Context: ctx,
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Fset: token.NewFileSet(),
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}

	combined := &ParseResult{}
	seen := make(map[string]bool) // seen file paths

	for _, pkg := range pkgs {
		// Collect non-fatal per-package errors; see ParseResult.LoadErrors.
		for _, e := range pkg.Errors {
			combined.LoadErrors = append(combined.LoadErrors, e)
		}

		pkgNode := model.Node{
			ID:   uuid.NewString(),
			Kind: model.KindPackage,
			Properties: map[string]any{
				model.PropName:       pkg.Name,
				model.PropImportPath: pkg.PkgPath,
				model.PropDir:        "",
			},
		}
		combined.Nodes = append(combined.Nodes, pkgNode)

		state := &parseState{
			typesByName: make(map[string]string),
		}

		for _, astFile := range pkg.Syntax {
			// Derive each file's path from the FileSet rather than pairing
			// pkg.Syntax with pkg.GoFiles by index — Syntax parallels
			// CompiledGoFiles, and the two lists diverge for generated or
			// cgo-processed files.
			tf := cfg.Fset.File(astFile.Pos())
			if tf == nil {
				combined.LoadErrors = append(combined.LoadErrors,
					fmt.Errorf("no file position for a syntax tree in %s", pkg.PkgPath))
				continue
			}
			filePath := tf.Name()
			if seen[filePath] {
				continue
			}
			seen[filePath] = true

			fset := cfg.Fset
			src, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", filePath, err)
			}

			fileID := uuid.NewString()
			fileNode := model.Node{
				ID:   fileID,
				Kind: model.KindFile,
				Properties: map[string]any{
					model.PropName:        filepath.Base(filePath),
					model.PropFilePath:    filePath,
					model.PropPackageName: pkg.Name,
					model.PropSource:      string(src),
				},
			}
			state.nodes = append(state.nodes, fileNode)

			// CONTAINS: Package → File
			state.edges = append(state.edges, model.Edge{
				Kind:       model.EdgeContains,
				SourceID:   pkgNode.ID,
				SourceKind: model.KindPackage,
				TargetID:   fileID,
				TargetKind: model.KindFile,
			})

			v := &visitor{
				fset:     fset,
				src:      src,
				fileID:   fileID,
				filePath: filePath,
				state:    state,
			}
			for _, decl := range astFile.Decls {
				v.visitDecl(decl)
			}
		}

		state.resolveDeferred()
		combined.Nodes = append(combined.Nodes, state.nodes...)
		combined.Edges = append(combined.Edges, state.edges...)
	}

	return combined, nil
}
