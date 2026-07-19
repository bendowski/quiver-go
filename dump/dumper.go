// Package dump writes Go source files from a property graph back to disk.
package dump

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/bendowski/quiver/model"
	"github.com/bendowski/quiver/store"
)

// Dumper reconstructs Go source files from a property graph and writes them
// to an output directory.
type Dumper struct {
	store store.Store
	fs    afero.Fs
}

// New creates a Dumper backed by the given Store and Afero filesystem.
func New(s store.Store, fs afero.Fs) *Dumper {
	return &Dumper{store: s, fs: fs}
}

// DumpAll writes every File stored in the graph to outDir. Output is
// flattened: each file is written directly to outDir under the basename of
// its file_path property, so files with the same basename overwrite each
// other.
func (d *Dumper) DumpAll(ctx context.Context, outDir string) error {
	files, err := d.store.GetNodesByKind(ctx, model.KindFile)
	if err != nil {
		return fmt.Errorf("DumpAll GetNodesByKind: %w", err)
	}
	for _, f := range files {
		if err := d.writeFile(ctx, f, outDir); err != nil {
			return err
		}
	}
	return nil
}

// DumpPackage writes only files belonging to the package with the given import
// path.
func (d *Dumper) DumpPackage(ctx context.Context, pkgPath, outDir string) error {
	pkgs, err := d.store.FindNodeByProperty(ctx, model.KindPackage, "import_path", pkgPath)
	if err != nil {
		return fmt.Errorf("DumpPackage find pkg: %w", err)
	}
	if len(pkgs) == 0 {
		return fmt.Errorf("package %q not found in store", pkgPath)
	}

	for _, pkg := range pkgs {
		edges, err := d.store.GetEdgesFrom(ctx, pkg.ID, model.KindPackage)
		if err != nil {
			return fmt.Errorf("DumpPackage GetEdgesFrom: %w", err)
		}
		for _, e := range edges {
			if e.Kind != model.EdgeContains || e.TargetKind != model.KindFile {
				continue
			}
			files, err := d.store.FindNodeByProperty(ctx, model.KindFile, "id", e.TargetID)
			if err != nil {
				return fmt.Errorf("DumpPackage find file %s: %w", e.TargetID, err)
			}
			if len(files) == 0 {
				continue // dangling CONTAINS edge; nothing to write
			}
			if err := d.writeFile(ctx, files[0], outDir); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeFile writes a single File node's source to outDir.
func (d *Dumper) writeFile(ctx context.Context, fileNode model.Node, outDir string) error {
	src, _ := fileNode.Properties["source"].(string)
	var content []byte

	if src != "" {
		content = []byte(src)
	} else {
		// Fallback: reconstruct from child nodes.
		var err error
		content, err = reconstructFile(ctx, d.store, fileNode)
		if err != nil {
			return fmt.Errorf("reconstruct %v: %w", fileNode.Properties["file_path"], err)
		}
	}

	filePath, _ := fileNode.Properties["file_path"].(string)
	if filePath == "" {
		name, _ := fileNode.Properties["name"].(string)
		if name == "" {
			name = fileNode.ID + ".go"
		}
		filePath = name
	}

	// TODO(#2): preserve the directory structure from file_path instead of
	// flattening; files with the same basename currently overwrite each other.
	outPath := filepath.Join(outDir, filepath.Base(filePath))
	if err := d.fs.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("MkdirAll: %w", err)
	}
	if err := afero.WriteFile(d.fs, outPath, content, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	return nil
}
