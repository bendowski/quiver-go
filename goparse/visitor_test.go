package goparse_test

import (
	"context"
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
