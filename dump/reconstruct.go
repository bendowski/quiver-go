package dump

import (
	"cmp"
	"context"
	"fmt"
	"go/format"
	"slices"
	"strings"

	"github.com/bendowski/quiver/model"
	"github.com/bendowski/quiver/store"
)

// reconstructFile reassembles a Go source file from its constituent nodes in
// the store. It is used as a fallback when the File node's "source" property
// is empty.
//
// The strategy:
//  1. Fetch the File node to get package_name.
//  2. Fetch all nodes contained in the file (via CONTAINS edges from the file).
//  3. Sort them by start_line, then emit their "source" property strings.
//  4. Format the result with go/format.Source, falling back to the
//     unformatted text when formatting fails.
//
// TODO(#3): Import snippets are stored as bare specs (`"fmt"`) without an
// enclosing import declaration, so files with imports reconstruct to invalid
// Go and always take the unformatted fallback.
func reconstructFile(ctx context.Context, s store.Store, fileNode model.Node) ([]byte, error) {
	pkgName, _ := fileNode.Properties[model.PropPackageName].(string)
	if pkgName == "" {
		pkgName = "main"
	}

	fileID := fileNode.ID

	edges, err := s.GetEdgesFrom(ctx, fileID, model.KindFile)
	if err != nil {
		return nil, fmt.Errorf("GetEdgesFrom file %s: %w", fileID, err)
	}

	type snippet struct {
		line   int64
		source string
	}
	var snippets []snippet

	for _, e := range edges {
		if e.Kind != model.EdgeContains {
			continue
		}
		switch e.TargetKind {
		case model.KindFunction, model.KindTypeDecl, model.KindVariable, model.KindImport:
		default:
			continue
		}
		nodes, err := s.FindNodeByProperty(ctx, e.TargetKind, model.PropID, e.TargetID)
		if err != nil {
			return nil, fmt.Errorf("find %s %s: %w", e.TargetKind, e.TargetID, err)
		}
		for _, n := range nodes {
			src, _ := n.Properties[model.PropSource].(string)
			line, _ := n.Properties[model.PropStartLine].(int64)
			if src != "" {
				snippets = append(snippets, snippet{line: line, source: src})
			}
		}
	}

	slices.SortFunc(snippets, func(a, b snippet) int {
		return cmp.Compare(a.line, b.line)
	})

	// The file is the package clause plus each snippet, separated by blank
	// lines.
	parts := make([]string, 0, len(snippets)+1)
	parts = append(parts, "package "+pkgName)
	for _, sn := range snippets {
		parts = append(parts, sn.source)
	}
	src := strings.Join(parts, "\n\n")

	formatted, err := format.Source([]byte(src))
	if err != nil {
		// Return the unformatted version rather than failing.
		return []byte(src), nil
	}
	return formatted, nil
}
