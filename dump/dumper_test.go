package dump_test

import (
	"context"
	"testing"

	"github.com/bendowski/quiver/dump"
	"github.com/bendowski/quiver/goparse"
	"github.com/bendowski/quiver/internal/testutil"
	"github.com/bendowski/quiver/model"
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

const dumpSrc = `package mypkg

func Hello() string {
	return "Hello"
}
`

func TestDumpAll_roundTrip(t *testing.T) {
	s := testutil.NewFakeStore()
	ctx := context.Background()

	// Parse into in-memory store.
	memfs := testutil.NewMemFs()
	testutil.WriteGoFile(t, memfs, "/src/hello.go", dumpSrc)

	parser := goparse.New(memfs)
	res, err := parser.ParseDir(ctx, "/src")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	for _, n := range res.Nodes {
		if err := s.AddNode(ctx, n); err != nil {
			t.Fatalf("AddNode %s: %v", n.ID, err)
		}
	}
	for _, e := range res.Edges {
		if err := s.AddEdge(ctx, e); err != nil {
			t.Fatalf("AddEdge: %v", err)
		}
	}

	// Dump to a separate in-memory FS.
	outfs := testutil.NewMemFs()
	d := dump.New(s, outfs)
	if err := d.DumpAll(ctx, "/out"); err != nil {
		t.Fatalf("DumpAll: %v", err)
	}

	// Verify the output file exists and contains the source.
	exists, err := afero.Exists(outfs, "/out/hello.go")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected /out/hello.go to exist after DumpAll")
	}

	got, err := afero.ReadFile(outfs, "/out/hello.go")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != dumpSrc {
		t.Errorf("file content mismatch\nwant: %s\n got: %s", dumpSrc, got)
	}
}

func TestDumpAll_emptyStore(t *testing.T) {
	s := testutil.NewFakeStore()
	ctx := context.Background()
	outfs := testutil.NewMemFs()
	d := dump.New(s, outfs)
	if err := d.DumpAll(ctx, "/out"); err != nil {
		t.Fatalf("DumpAll on empty store: %v", err)
	}
}

func TestDumpAll_reconstructFallback(t *testing.T) {
	// Create a File node with no "source" to force reconstruction path.
	s := testutil.NewFakeStore()
	ctx := context.Background()

	pkgID := uuid.NewString()
	fileID := uuid.NewString()

	if err := s.AddNode(ctx, model.Node{
		ID:   pkgID,
		Kind: model.KindPackage,
		Properties: map[string]any{
			"name": "fallback", "import_path": "fallback", "dir": "/fb",
		},
	}); err != nil {
		t.Fatalf("AddNode pkg: %v", err)
	}

	// File with empty source.
	if err := s.AddNode(ctx, model.Node{
		ID:   fileID,
		Kind: model.KindFile,
		Properties: map[string]any{
			"name":         "fb.go",
			"file_path":    "/fb/fb.go",
			"package_name": "fallback",
			"source":       "",
		},
	}); err != nil {
		t.Fatalf("AddNode file: %v", err)
	}

	if err := s.AddEdge(ctx, model.Edge{
		Kind:       model.EdgeContains,
		SourceID:   pkgID,
		SourceKind: model.KindPackage,
		TargetID:   fileID,
		TargetKind: model.KindFile,
	}); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}

	outfs := testutil.NewMemFs()
	d := dump.New(s, outfs)
	// Should not error even with empty source – it falls back to reconstruct.
	if err := d.DumpAll(ctx, "/out"); err != nil {
		t.Fatalf("DumpAll with fallback: %v", err)
	}
}
