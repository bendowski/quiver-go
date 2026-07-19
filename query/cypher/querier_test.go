package cypher_test

import (
	"context"
	"testing"

	"github.com/bendowski/quiver/internal/dbtest"
	"github.com/bendowski/quiver/model"
	"github.com/bendowski/quiver/query/cypher"
	"github.com/google/uuid"
)

func TestCypherQuerier_Query(t *testing.T) {
	s := dbtest.NewMemStore(t)
	ctx := context.Background()

	// Seed one Package node directly via the Store.
	pkgID := uuid.NewString()
	if err := s.AddNode(ctx, model.Node{
		ID:   pkgID,
		Kind: model.KindPackage,
		Properties: map[string]any{
			"name":        "testpkg",
			"import_path": "github.com/example/testpkg",
			"dir":         "/tmp/testpkg",
		},
	}); err != nil {
		t.Fatalf("AddNode: %v", err)
	}

	q := cypher.New(s.Connection())
	result, err := q.Query(ctx, "MATCH (n:Package) RETURN n.name")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(result.Rows))
	}
	if result.Rows[0]["n.name"] != "testpkg" {
		t.Errorf("n.name: want testpkg, got %v", result.Rows[0]["n.name"])
	}
}

func TestCypherQuerier_QueryWithParams(t *testing.T) {
	s := dbtest.NewMemStore(t)
	ctx := context.Background()

	pkgID := uuid.NewString()
	if err := s.AddNode(ctx, model.Node{
		ID:   pkgID,
		Kind: model.KindPackage,
		Properties: map[string]any{
			"name":        "paramspkg",
			"import_path": "github.com/example/paramspkg",
			"dir":         "/tmp/paramspkg",
		},
	}); err != nil {
		t.Fatalf("AddNode: %v", err)
	}

	q := cypher.New(s.Connection())
	result, err := q.QueryWithParams(ctx,
		"MATCH (n:Package) WHERE n.name = $pkg_name RETURN n.import_path",
		map[string]any{"pkg_name": "paramspkg"},
	)
	if err != nil {
		t.Fatalf("QueryWithParams: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(result.Rows))
	}
	if result.Rows[0]["n.import_path"] != "github.com/example/paramspkg" {
		t.Errorf("import_path: unexpected %v", result.Rows[0]["n.import_path"])
	}
}
