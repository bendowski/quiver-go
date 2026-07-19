package ladybug_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/bendowski/quiver/internal/dbtest"
	"github.com/bendowski/quiver/model"
)

func TestAddNodeAndGetByKind(t *testing.T) {
	s := dbtest.NewMemStore(t)
	ctx := context.Background()

	node := model.Node{
		ID:   uuid.NewString(),
		Kind: model.KindPackage,
		Properties: map[string]any{
			"name":        "mypkg",
			"import_path": "github.com/example/mypkg",
			"dir":         "/some/dir",
		},
	}
	if err := s.AddNode(ctx, node); err != nil {
		t.Fatalf("AddNode: %v", err)
	}

	nodes, err := s.GetNodesByKind(ctx, model.KindPackage)
	if err != nil {
		t.Fatalf("GetNodesByKind: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("want 1 Package node, got %d", len(nodes))
	}
	got := nodes[0]
	if got.ID != node.ID {
		t.Errorf("ID: want %s, got %s", node.ID, got.ID)
	}
	if got.Properties["name"] != "mypkg" {
		t.Errorf("name: want mypkg, got %v", got.Properties["name"])
	}
}

func TestAddEdgeAndGetEdgesFrom(t *testing.T) {
	s := dbtest.NewMemStore(t)
	ctx := context.Background()

	pkgID := uuid.NewString()
	fileID := uuid.NewString()

	pkg := model.Node{
		ID:   pkgID,
		Kind: model.KindPackage,
		Properties: map[string]any{
			"name":        "mypkg",
			"import_path": "github.com/example/mypkg",
			"dir":         "/some/dir",
		},
	}
	file := model.Node{
		ID:   fileID,
		Kind: model.KindFile,
		Properties: map[string]any{
			"name":         "foo.go",
			"file_path":    "/some/dir/foo.go",
			"package_name": "mypkg",
			"source":       "package mypkg\n",
		},
	}
	if err := s.AddNode(ctx, pkg); err != nil {
		t.Fatalf("AddNode pkg: %v", err)
	}
	if err := s.AddNode(ctx, file); err != nil {
		t.Fatalf("AddNode file: %v", err)
	}

	edge := model.Edge{
		Kind:       model.EdgeContains,
		SourceID:   pkgID,
		SourceKind: model.KindPackage,
		TargetID:   fileID,
		TargetKind: model.KindFile,
	}
	if err := s.AddEdge(ctx, edge); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}

	edges, err := s.GetEdgesFrom(ctx, pkgID, model.KindPackage)
	if err != nil {
		t.Fatalf("GetEdgesFrom: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("want 1 edge, got %d", len(edges))
	}
	got := edges[0]
	if got.Kind != model.EdgeContains {
		t.Errorf("Kind: want %s, got %s", model.EdgeContains, got.Kind)
	}
	if got.TargetID != fileID {
		t.Errorf("TargetID: want %s, got %s", fileID, got.TargetID)
	}
	if got.TargetKind != model.KindFile {
		t.Errorf("TargetKind: want %s, got %s", model.KindFile, got.TargetKind)
	}
}

func TestFindNodeByProperty(t *testing.T) {
	s := dbtest.NewMemStore(t)
	ctx := context.Background()

	n := model.Node{
		ID:   uuid.NewString(),
		Kind: model.KindFunction,
		Properties: map[string]any{
			"name":        "DoThing",
			"signature":   "DoThing() error",
			"receiver":    "",
			"source":      "func DoThing() error { return nil }",
			"file_path":   "/foo.go",
			"start_line":  int64(1),
			"end_line":    int64(1),
			"doc_comment": "",
			"exported":    true,
		},
	}
	if err := s.AddNode(ctx, n); err != nil {
		t.Fatalf("AddNode: %v", err)
	}

	found, err := s.FindNodeByProperty(ctx, model.KindFunction, "name", "DoThing")
	if err != nil {
		t.Fatalf("FindNodeByProperty: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("want 1 node, got %d", len(found))
	}
	if found[0].ID != n.ID {
		t.Errorf("ID mismatch: want %s, got %s", n.ID, found[0].ID)
	}
}

func TestClear(t *testing.T) {
	s := dbtest.NewMemStore(t)
	ctx := context.Background()

	n := model.Node{
		ID:   uuid.NewString(),
		Kind: model.KindPackage,
		Properties: map[string]any{
			"name":        "x",
			"import_path": "x",
			"dir":         "/x",
		},
	}
	if err := s.AddNode(ctx, n); err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	if err := s.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	nodes, err := s.GetNodesByKind(ctx, model.KindPackage)
	if err != nil {
		t.Fatalf("GetNodesByKind after Clear: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("want 0 nodes after Clear, got %d", len(nodes))
	}
}
