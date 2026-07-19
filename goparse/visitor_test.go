package goparse_test

import (
	"context"
	"slices"
	"testing"

	"github.com/bendowski/quiver/goparse"
	"github.com/bendowski/quiver/internal/testutil"
	"github.com/bendowski/quiver/model"
)

const embedSrc = `package embed

type Base struct {
	X int
}

type Child struct {
	Base
	Y string
}
`

func TestVisitor_embedsEdge(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/embed.go", embedSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	var embedsCount int
	for _, e := range res.Edges {
		if e.Kind == model.EdgeEmbeds {
			embedsCount++
		}
	}
	if embedsCount != 1 {
		t.Errorf("want 1 EMBEDS edge, got %d", embedsCount)
	}
}

const qualifiedEmbedSrc = `package embed

import "sync"

type Guarded struct {
	sync.Mutex
	N int
}
`

func TestVisitor_qualifiedEmbedFieldName(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/embed.go", qualifiedEmbedSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	var names []string
	for _, n := range res.Nodes {
		if n.Kind == model.KindField {
			name, _ := n.Properties[model.PropName].(string)
			names = append(names, name)
		}
	}
	if !slices.Contains(names, "Mutex") {
		t.Errorf("want an embedded field named Mutex, got %v", names)
	}
}

const unicodeFieldSrc = `package uni

type S struct {
	Łódź string
	żaba string
}
`

func TestVisitor_unicodeExportedField(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/uni.go", unicodeFieldSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	want := map[string]bool{"Łódź": true, "żaba": false}
	for _, n := range res.Nodes {
		if n.Kind != model.KindField {
			continue
		}
		name, _ := n.Properties[model.PropName].(string)
		wantExported, ok := want[name]
		if !ok {
			continue
		}
		if exported, _ := n.Properties[model.PropExported].(bool); exported != wantExported {
			t.Errorf("field %s: exported want %v, got %v", name, wantExported, exported)
		}
		delete(want, name)
	}
	for name := range want {
		t.Errorf("field %s not found", name)
	}
}

const groupedDocSrc = `package grp

type (
	// A is documented on the spec.
	A struct{}
	// B has its own doc too.
	B struct{}
)
`

func TestVisitor_groupedTypeSpecDocs(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/grp.go", groupedDocSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	want := map[string]string{
		"A": "A is documented on the spec.",
		"B": "B has its own doc too.",
	}
	for _, n := range res.Nodes {
		if n.Kind != model.KindTypeDecl {
			continue
		}
		name, _ := n.Properties[model.PropName].(string)
		if doc, _ := n.Properties[model.PropDocComment].(string); doc != want[name] {
			t.Errorf("%s doc: want %q, got %q", name, want[name], doc)
		}
	}
}

const interfaceSrc = `package iface

type Stringer interface {
	String() string
}
`

func TestVisitor_interfaceTypeDecl(t *testing.T) {
	fs := testutil.NewMemFs()
	testutil.WriteGoFile(t, fs, "/pkg/iface.go", interfaceSrc)

	p := goparse.New(fs)
	res, err := p.ParseDir(context.Background(), "/pkg")
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}

	var found *model.Node
	for i, n := range res.Nodes {
		if n.Kind == model.KindTypeDecl && n.Properties["name"] == "Stringer" {
			found = &res.Nodes[i]
			break
		}
	}
	if found == nil {
		t.Fatal("Stringer TypeDecl not found")
	}
	if found.Properties["kind"] != "interface" {
		t.Errorf("kind: want interface, got %v", found.Properties["kind"])
	}
}
