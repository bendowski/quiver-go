package dump

import (
	"context"
	"fmt"
	"go/format"
	"sort"
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
	pkgName, _ := fileNode.Properties["package_name"].(string)
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
		var nodes []model.Node
		for _, kind := range []model.NodeKind{
			model.KindFunction, model.KindTypeDecl, model.KindVariable, model.KindImport,
		} {
			if e.TargetKind != kind {
				continue
			}
			nodes, err = s.FindNodeByProperty(ctx, kind, "id", e.TargetID)
			if err != nil {
				continue
			}
			break
		}
		for _, n := range nodes {
			src, _ := n.Properties["source"].(string)
			line, _ := n.Properties["start_line"].(int64)
			if src != "" {
				snippets = append(snippets, snippet{line: line, source: src})
			}
		}
	}

	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].line < snippets[j].line
	})

	var sb strings.Builder
	sb.WriteString("package ")
	sb.WriteString(pkgName)
	sb.WriteString("\n\n")
	for i, s := range snippets {
		sb.WriteString(s.source)
		if i < len(snippets)-1 {
			sb.WriteString("\n\n")
		}
	}

	src := sb.String()
	formatted, err := format.Source([]byte(src))
	if err != nil {
		// Return the unformatted version rather than failing.
		return []byte(src), nil
	}
	return formatted, nil
}
