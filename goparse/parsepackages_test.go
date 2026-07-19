package goparse_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"

	"github.com/bendowski/quiver/goparse"
	"github.com/bendowski/quiver/model"
)

// setupModule creates a temp module with the given files and makes it the
// working directory for the test. ParsePackages needs a real filesystem and
// the go toolchain (packages.Load shells out to `go list`).
func setupModule(t *testing.T, files map[string]string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/fixture\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	t.Chdir(dir)
}

func TestParsePackages_structuralNodes(t *testing.T) {
	setupModule(t, map[string]string{
		"a.go": "package fixture\n\n// T is a type.\ntype T struct{ F int }\n\nfunc Fn() {}\n",
	})

	res, err := goparse.New(afero.NewOsFs()).ParsePackages(context.Background(), ".")
	if err != nil {
		t.Fatalf("ParsePackages: %v", err)
	}
	if len(res.LoadErrors) != 0 {
		t.Fatalf("want no load errors, got %v", res.LoadErrors)
	}

	counts := make(map[model.NodeKind]int)
	for _, n := range res.Nodes {
		counts[n.Kind]++
	}
	for kind, want := range map[model.NodeKind]int{
		model.KindPackage:  1,
		model.KindFile:     1,
		model.KindTypeDecl: 1,
		model.KindField:    1,
		model.KindFunction: 1,
	} {
		if counts[kind] != want {
			t.Errorf("kind %s: want %d, got %d", kind, want, counts[kind])
		}
	}

	for _, n := range res.Nodes {
		if n.Kind == model.KindFile && n.Properties[model.PropPackageName] != "fixture" {
			t.Errorf("File package_name: want fixture, got %v", n.Properties[model.PropPackageName])
		}
	}
}

func TestParsePackages_collectsLoadErrors(t *testing.T) {
	setupModule(t, map[string]string{
		"broken.go": "package fixture\n\nfunc F() int { return \"not an int\" }\n",
	})

	res, err := goparse.New(afero.NewOsFs()).ParsePackages(context.Background(), ".")
	if err != nil {
		t.Fatalf("ParsePackages: %v", err)
	}
	if len(res.LoadErrors) == 0 {
		t.Fatal("want load errors for a type-broken package, got none")
	}
	if len(res.Nodes) == 0 {
		t.Fatal("want structural nodes despite type errors, got none")
	}
}
