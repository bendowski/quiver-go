package cypher_test

import (
	"strings"
	"testing"

	"github.com/bendowski/quiver/model"
	"github.com/bendowski/quiver/query/cypher"
)

func TestBuildCreateNode(t *testing.T) {
	stmt, params := cypher.BuildCreateNode(
		model.KindFunction,
		"uuid-1",
		map[string]any{
			"name":     "Foo",
			"exported": true,
		},
	)

	if !strings.HasPrefix(stmt, "CREATE (n:Function {") {
		t.Errorf("unexpected statement prefix: %s", stmt)
	}
	if !strings.Contains(stmt, "id: $p_id") {
		t.Errorf("id param missing in statement: %s", stmt)
	}
	if params["p_id"] != "uuid-1" {
		t.Errorf("p_id: want uuid-1, got %v", params["p_id"])
	}
	if params["p_name"] != "Foo" {
		t.Errorf("p_name: want Foo, got %v", params["p_name"])
	}
	if params["p_exported"] != true {
		t.Errorf("p_exported: want true, got %v", params["p_exported"])
	}
}

func TestBuildCreateEdge_noProps(t *testing.T) {
	stmt, params := cypher.BuildCreateEdge(
		model.KindPackage, "src-id",
		model.KindFile, "dst-id",
		model.EdgeContains,
		nil,
	)
	want := "MATCH (a:Package {id: $p_src}), (b:File {id: $p_dst}) CREATE (a)-[:CONTAINS]->(b)"
	if stmt != want {
		t.Errorf("stmt:\nwant: %s\n got: %s", want, stmt)
	}
	if params["p_src"] != "src-id" {
		t.Errorf("p_src mismatch")
	}
	if params["p_dst"] != "dst-id" {
		t.Errorf("p_dst mismatch")
	}
}

func TestBuildCreateEdge_withProps(t *testing.T) {
	stmt, params := cypher.BuildCreateEdge(
		model.KindFunction, "a",
		model.KindFunction, "b",
		model.EdgeCalls,
		map[string]any{"line": int64(42), "file_path": "/foo.go"},
	)
	if !strings.Contains(stmt, "ep_file_path") {
		t.Errorf("file_path param missing: %s", stmt)
	}
	if !strings.Contains(stmt, "ep_line") {
		t.Errorf("line param missing: %s", stmt)
	}
	if params["ep_line"] != int64(42) {
		t.Errorf("ep_line: want 42, got %v", params["ep_line"])
	}
}

func TestBuildMatchByKind(t *testing.T) {
	stmt := cypher.BuildMatchByKind(model.KindTypeDecl)
	if stmt != "MATCH (n:TypeDecl) RETURN n" {
		t.Errorf("unexpected: %s", stmt)
	}
}

func TestBuildMatchByProperty(t *testing.T) {
	stmt, params := cypher.BuildMatchByProperty(model.KindFunction, "name", "Bar")
	if !strings.Contains(stmt, "WHERE n.name = $p_val") {
		t.Errorf("unexpected stmt: %s", stmt)
	}
	if params["p_val"] != "Bar" {
		t.Errorf("p_val: want Bar, got %v", params["p_val"])
	}
}

func TestBuildMatchByProperty_invalidPropPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for non-identifier property name")
		}
	}()
	cypher.BuildMatchByProperty(model.KindFunction, "name = '' OR 1=1 //", "x")
}

func TestBuildMatchEdgesFrom(t *testing.T) {
	stmt, params := cypher.BuildMatchEdgesFrom(model.KindFile, "file-id")
	if !strings.Contains(stmt, "MATCH (a:File {id: $p_id})-[r]->(b)") {
		t.Errorf("unexpected stmt: %s", stmt)
	}
	if params["p_id"] != "file-id" {
		t.Errorf("p_id mismatch")
	}
}
