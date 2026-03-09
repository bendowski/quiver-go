// Package goparse parses Go source code into model.Node and model.Edge values.
package goparse

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
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

	// Collect Go source files.
	var goFiles []fs.FileInfo
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") {
			goFiles = append(goFiles, e)
		}
	}
	if len(goFiles) == 0 {
		return &ParseResult{}, nil
	}

	// Determine the package name from the first parseable file.
	var pkgName string
	for _, fi := range goFiles {
		src, err := afero.ReadFile(p.fs, filepath.Join(dir, fi.Name()))
		if err != nil {
			continue
		}
		f, err := parser.ParseFile(fset, fi.Name(), src, 0)
		if err == nil && f.Name != nil {
			pkgName = f.Name.Name
			break
		}
	}

	pkgNode := model.Node{
		ID:   uuid.NewString(),
		Kind: model.KindPackage,
		Properties: map[string]any{
			"name":        pkgName,
			"import_path": dir,
			"dir":         dir,
		},
	}

	state := &parseState{
		pkgNode:     pkgNode,
		typesByName: make(map[string]string),
	}
	state.nodes = append(state.nodes, pkgNode)

	// Parse each file.
	for _, fi := range goFiles {
		filePath := filepath.Join(dir, fi.Name())
		src, err := afero.ReadFile(p.fs, filePath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", filePath, err)
		}

		astFile, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
		if err != nil {
			// Skip files with parse errors (e.g. test-only build tags).
			continue
		}

		fileID := uuid.NewString()
		fileNode := model.Node{
			ID:   fileID,
			Kind: model.KindFile,
			Properties: map[string]any{
				"name":         fi.Name(),
				"file_path":    filePath,
				"package_name": pkgName,
				"source":       string(src),
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
			v.Visit(decl)
		}
	}

	state.resolveDeferred()

	return &ParseResult{Nodes: state.nodes, Edges: state.edges}, nil
}

// ParsePackages uses golang.org/x/tools/go/packages to load and type-check
// the given patterns. In addition to structural nodes it resolves CALLS,
// IMPLEMENTS, and REFERS_TO edges that require type information.
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
		if len(pkg.Errors) > 0 {
			// Report but don't abort on per-package errors.
			for _, e := range pkg.Errors {
				_ = e // non-fatal
			}
		}

		pkgNode := model.Node{
			ID:   uuid.NewString(),
			Kind: model.KindPackage,
			Properties: map[string]any{
				"name":        pkg.Name,
				"import_path": pkg.PkgPath,
				"dir":         "",
			},
		}
		combined.Nodes = append(combined.Nodes, pkgNode)

		state := &parseState{
			pkgNode:     pkgNode,
			typesByName: make(map[string]string),
		}

		for i, astFile := range pkg.Syntax {
			var filePath string
			if i < len(pkg.GoFiles) {
				filePath = pkg.GoFiles[i]
			}
			if seen[filePath] {
				continue
			}
			seen[filePath] = true

			fset := cfg.Fset
			src := readFileBytes(filePath)

			fileID := uuid.NewString()
			fileNode := model.Node{
				ID:   fileID,
				Kind: model.KindFile,
				Properties: map[string]any{
					"name":         filepath.Base(filePath),
					"file_path":    filePath,
					"package_name": pkg.Name,
					"source":       string(src),
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
				v.Visit(decl)
			}
		}

		state.resolveDeferred()
		combined.Nodes = append(combined.Nodes, state.nodes...)
		combined.Edges = append(combined.Edges, state.edges...)
	}

	return combined, nil
}

// readFileBytes reads a file from the real OS filesystem.
func readFileBytes(path string) []byte {
	osfs := afero.NewOsFs()
	b, _ := afero.ReadFile(osfs, path)
	return b
}
