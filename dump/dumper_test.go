package dump_test

import (
	"context"
	"go/parser"
	"go/token"
	"strings"
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

	// A contained function gives the reconstruction real content. (Imports
	// are deliberately absent: reconstructed import blocks are issue #3.)
	fnID := uuid.NewString()
	if err := s.AddNode(ctx, model.Node{
		ID:   fnID,
		Kind: model.KindFunction,
		Properties: map[string]any{
			model.PropName:      "Hello",
			model.PropSource:    "func Hello() string {\n\treturn \"hi\"\n}",
			model.PropStartLine: int64(3),
		},
	}); err != nil {
		t.Fatalf("AddNode fn: %v", err)
	}
	if err := s.AddEdge(ctx, model.Edge{
		Kind:       model.EdgeContains,
		SourceID:   fileID,
		SourceKind: model.KindFile,
		TargetID:   fnID,
		TargetKind: model.KindFunction,
	}); err != nil {
		t.Fatalf("AddEdge fn: %v", err)
	}

	outfs := testutil.NewMemFs()
	d := dump.New(s, outfs)
	if err := d.DumpAll(ctx, "/out"); err != nil {
		t.Fatalf("DumpAll with fallback: %v", err)
	}

	got, err := afero.ReadFile(outfs, "/out/fb.go")
	if err != nil {
		t.Fatalf("ReadFile reconstructed output: %v", err)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "fb.go", got, 0); err != nil {
		t.Errorf("reconstructed output is not valid Go: %v\n%s", err, got)
	}
	if !strings.Contains(string(got), "func Hello()") {
		t.Errorf("reconstructed output missing function:\n%s", got)
	}
}

// seedPackageWithFile inserts a Package containing one File with source.
func seedPackageWithFile(t *testing.T, s *testutil.FakeStore, importPath, fileName, source string) {
	t.Helper()
	ctx := context.Background()
	pkgID := uuid.NewString()
	fileID := uuid.NewString()
	if err := s.AddNode(ctx, model.Node{
		ID:   pkgID,
		Kind: model.KindPackage,
		Properties: map[string]any{
			model.PropName:       importPath,
			model.PropImportPath: importPath,
			model.PropDir:        "/",
		},
	}); err != nil {
		t.Fatalf("AddNode pkg: %v", err)
	}
	if err := s.AddNode(ctx, model.Node{
		ID:   fileID,
		Kind: model.KindFile,
		Properties: map[string]any{
			model.PropName:        fileName,
			model.PropFilePath:    "/" + fileName,
			model.PropPackageName: importPath,
			model.PropSource:      source,
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
}

func TestDumpPackage(t *testing.T) {
	s := testutil.NewFakeStore()
	seedPackageWithFile(t, s, "example.com/one", "one.go", "package one\n")
	seedPackageWithFile(t, s, "example.com/two", "two.go", "package two\n")

	outfs := testutil.NewMemFs()
	d := dump.New(s, outfs)
	if err := d.DumpPackage(context.Background(), "example.com/one", "/out"); err != nil {
		t.Fatalf("DumpPackage: %v", err)
	}

	if ok, _ := afero.Exists(outfs, "/out/one.go"); !ok {
		t.Error("expected /out/one.go to be written")
	}
	if ok, _ := afero.Exists(outfs, "/out/two.go"); ok {
		t.Error("two.go belongs to another package and must not be written")
	}
}

func TestDumpPackage_unknownPackage(t *testing.T) {
	s := testutil.NewFakeStore()
	d := dump.New(s, testutil.NewMemFs())
	if err := d.DumpPackage(context.Background(), "example.com/missing", "/out"); err == nil {
		t.Fatal("expected an error for an unknown package import path")
	}
}
