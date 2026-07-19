package schema

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/bendowski/quiver/goparse"
	"github.com/bendowski/quiver/internal/testutil"
	"github.com/bendowski/quiver/model"
)

// consistencySrc produces at least one node of every kind: Package and File
// implicitly, plus Import, TypeDecl, Field, Function, and Variable.
const consistencySrc = `package fixture

import "fmt"

// T is a struct.
type T struct {
	F int
}

// V is a variable.
var V = fmt.Sprint("x")

// Fn is a function.
func Fn(s string) string { return s }
`

var nodeTableRE = regexp.MustCompile(`^CREATE NODE TABLE (\w+)\((.+)\)$`)

// ddlColumns parses nodeTableDDL into table name → set of column names.
func ddlColumns(t *testing.T) map[string]map[string]bool {
	t.Helper()
	tables := make(map[string]map[string]bool, len(nodeTableDDL))
	for _, stmt := range nodeTableDDL {
		m := nodeTableRE.FindStringSubmatch(stmt)
		if m == nil {
			t.Fatalf("cannot parse DDL statement: %s", stmt)
		}
		cols := make(map[string]bool)
		for _, col := range strings.Split(m[2], ", ") {
			cols[strings.Fields(col)[0]] = true
		}
		tables[m[1]] = cols
	}
	return tables
}

// TestSchemaMatchesParserProperties locks the DDL columns to the property
// keys goparse actually emits: every emitted key must be a column, and every
// column must be emitted. The one exception is "id", which parsed nodes carry
// in Node.ID and the store injects as a property on insert.
func TestSchemaMatchesParserProperties(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/fixture.go", consistencySrc)

	res, err := goparse.New(fs).ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	propsByKind := make(map[model.NodeKind]map[string]bool)
	for _, n := range res.Nodes {
		if propsByKind[n.Kind] == nil {
			propsByKind[n.Kind] = make(map[string]bool)
		}
		for k := range n.Properties {
			propsByKind[n.Kind][k] = true
		}
	}

	tables := ddlColumns(t)

	for kind, cols := range tables {
		emitted := propsByKind[model.NodeKind(kind)]
		if emitted == nil {
			t.Errorf("fixture produced no %s node; extend consistencySrc", kind)
			continue
		}
		for k := range emitted {
			if !cols[k] {
				t.Errorf("%s: parser emits property %q but the DDL has no such column", kind, k)
			}
		}
		for c := range cols {
			if c != model.PropID && !emitted[c] {
				t.Errorf("%s: DDL declares column %q but the parser never emits it", kind, c)
			}
		}
	}

	for kind := range propsByKind {
		if tables[string(kind)] == nil {
			t.Errorf("parser produced %s nodes but the DDL has no such table", kind)
		}
	}
}
