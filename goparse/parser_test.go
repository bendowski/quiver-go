package goparse_test

import (
	"context"
	"testing"

	"github.com/bendowski/quiver/goparse"
	"github.com/bendowski/quiver/internal/testutil"
	"github.com/bendowski/quiver/model"
)

const simpleSrc = `package testpkg

import "fmt"

// Greeter greets people.
type Greeter struct {
	Name string
	Age  int
}

// Hello greets.
func (g *Greeter) Hello() string {
	return fmt.Sprintf("Hello, %s!", g.Name)
}

func Greet(name string) string {
	return fmt.Sprintf("Hi, %s!", name)
}

func hidden() {}

const DefaultName = "World"

var greeting = "Hello"
`

func TestParseDir_nodeKinds(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/simple.go", simpleSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	counts := make(map[model.NodeKind]int)
	for _, n := range res.Nodes {
		counts[n.Kind]++
	}

	tests := []struct {
		kind model.NodeKind
		want int
	}{
		{model.KindPackage, 1},
		{model.KindFile, 1},
		{model.KindImport, 1},
		{model.KindTypeDecl, 1},
		{model.KindFunction, 3}, // Hello, Greet, hidden
		{model.KindField, 2},    // Name, Age
		{model.KindVariable, 2}, // DefaultName, greeting
	}
	for _, tc := range tests {
		if counts[tc.kind] != tc.want {
			t.Errorf("kind %s: want %d, got %d", tc.kind, tc.want, counts[tc.kind])
		}
	}
}

func TestParseDir_containsEdges(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/simple.go", simpleSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	edgeCounts := make(map[model.EdgeKind]int)
	for _, e := range res.Edges {
		edgeCounts[e.Kind]++
	}

	if edgeCounts[model.EdgeContains] == 0 {
		t.Error("expected CONTAINS edges, got none")
	}
	if edgeCounts[model.EdgeImports] == 0 {
		t.Error("expected IMPORTS edges, got none")
	}
}

func TestParseDir_hasReceiverEdge(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/simple.go", simpleSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	var hasReceiverCount int
	for _, e := range res.Edges {
		if e.Kind == model.EdgeHasReceiver {
			hasReceiverCount++
		}
	}
	if hasReceiverCount != 1 {
		t.Errorf("want 1 HAS_RECEIVER edge, got %d", hasReceiverCount)
	}
}

func TestParseDir_functionProperties(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/simple.go", simpleSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	var greet *model.Node
	for i, n := range res.Nodes {
		if n.Kind == model.KindFunction && n.Properties["name"] == "Greet" {
			greet = &res.Nodes[i]
			break
		}
	}
	if greet == nil {
		t.Fatal("Greet function not found")
	}
	if greet.Properties["exported"] != true {
		t.Errorf("Greet.exported: want true, got %v", greet.Properties["exported"])
	}

	var hidden *model.Node
	for i, n := range res.Nodes {
		if n.Kind == model.KindFunction && n.Properties["name"] == "hidden" {
			hidden = &res.Nodes[i]
			break
		}
	}
	if hidden == nil {
		t.Fatal("hidden function not found")
	}
	if hidden.Properties["exported"] != false {
		t.Errorf("hidden.exported: want false, got %v", hidden.Properties["exported"])
	}
}

func TestParseDir_mixedPackageNames(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/a.go", "package x\n\nfunc A() {}\n")
	testutil.WriteGoFile(t, fs, "/pkg/b.go", "package x_test\n\nfunc B() {}\n")

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	got := make(map[string]string) // file name → package_name
	for _, n := range res.Nodes {
		if n.Kind == model.KindFile {
			name, _ := n.Properties[model.PropName].(string)
			pkg, _ := n.Properties[model.PropPackageName].(string)
			got[name] = pkg
		}
	}
	for name, want := range map[string]string{"a.go": "x", "b.go": "x_test"} {
		if got[name] != want {
			t.Errorf("%s: package_name want %q, got %q", name, want, got[name])
		}
	}
}

func TestParseDir_emptyDir(t *testing.T) {
	fs := testutil.NewMemFs()
	_ = fs.MkdirAll("/empty", 0o755)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/empty")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if len(res.Nodes) != 0 {
		t.Errorf("want 0 nodes for empty dir, got %d", len(res.Nodes))
	}
}
