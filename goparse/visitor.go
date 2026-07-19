package goparse

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"github.com/bendowski/quiver/model"
	"github.com/google/uuid"
)

// visitor walks a single Go source file's AST and populates nodes and edges in
// a parseState. One visitor is created per file.
type visitor struct {
	fset     *token.FileSet
	src      []byte
	fileID   string
	filePath string
	state    *parseState
}

// parseState accumulates nodes, edges, and metadata across files in a package.
type parseState struct {
	nodes   []model.Node
	edges   []model.Edge
	pkgNode model.Node // the single Package node for this parse run
	// typesByName maps unqualified type name → node ID for HAS_RECEIVER / EMBEDS resolution.
	typesByName map[string]string
	// unresolvedReceivers holds (funcID, receiverTypeName) pairs for post-pass resolution.
	unresolvedReceivers []unresolvedReceiver
	// unresolvedEmbeds holds (typeDeclID, embeddedTypeName) pairs.
	unresolvedEmbeds []unresolvedEmbed
}

type unresolvedReceiver struct {
	funcID   string
	typeName string
}

type unresolvedEmbed struct {
	typeDeclID string
	typeName   string
}

// Visit implements ast.Visitor.
func (v *visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.FuncDecl:
		v.handleFuncDecl(n)
		return nil // do not recurse into function body
	case *ast.GenDecl:
		v.handleGenDecl(n)
		return nil
	}
	return v
}

func (v *visitor) handleFuncDecl(fn *ast.FuncDecl) {
	if fn.Name == nil {
		return
	}
	id := uuid.NewString()
	pos := v.fset.Position(fn.Pos())
	end := v.fset.Position(fn.End())

	receiver := extractReceiverTypeName(fn.Recv)
	sig := fn.Name.Name + formatFuncType(v.fset, fn.Type)

	node := model.Node{
		ID:   id,
		Kind: model.KindFunction,
		Properties: map[string]any{
			"name":        fn.Name.Name,
			"signature":   sig,
			"receiver":    receiver,
			"source":      v.sourceOf(fn),
			"file_path":   v.filePath,
			"start_line":  int64(pos.Line),
			"end_line":    int64(end.Line),
			"doc_comment": docText(fn.Doc),
			"exported":    fn.Name.IsExported(),
		},
	}
	v.state.nodes = append(v.state.nodes, node)

	// CONTAINS: File → Function
	v.state.edges = append(v.state.edges, model.Edge{
		Kind:       model.EdgeContains,
		SourceID:   v.fileID,
		SourceKind: model.KindFile,
		TargetID:   id,
		TargetKind: model.KindFunction,
	})

	// Defer HAS_RECEIVER resolution until all TypeDecls are known.
	if receiver != "" {
		v.state.unresolvedReceivers = append(v.state.unresolvedReceivers, unresolvedReceiver{
			funcID:   id,
			typeName: receiver,
		})
	}
}

func (v *visitor) handleGenDecl(decl *ast.GenDecl) {
	switch decl.Tok {
	case token.IMPORT:
		for _, spec := range decl.Specs {
			is, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			v.handleImport(decl, is)
		}
	case token.TYPE:
		for _, spec := range decl.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			v.handleTypeSpec(decl, ts)
		}
	case token.VAR, token.CONST:
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			v.handleValueSpec(decl, vs)
		}
	}
}

func (v *visitor) handleImport(decl *ast.GenDecl, spec *ast.ImportSpec) {
	id := uuid.NewString()
	pos := v.fset.Position(spec.Pos())
	end := v.fset.Position(spec.End())

	alias := ""
	if spec.Name != nil {
		alias = spec.Name.Name
	}
	path := strings.Trim(spec.Path.Value, `"`)

	node := model.Node{
		ID:   id,
		Kind: model.KindImport,
		Properties: map[string]any{
			"alias":      alias,
			"path":       path,
			"file_path":  v.filePath,
			"start_line": int64(pos.Line),
			"end_line":   int64(end.Line),
		},
	}
	v.state.nodes = append(v.state.nodes, node)

	// IMPORTS: File → Import
	v.state.edges = append(v.state.edges, model.Edge{
		Kind:       model.EdgeImports,
		SourceID:   v.fileID,
		SourceKind: model.KindFile,
		TargetID:   id,
		TargetKind: model.KindImport,
	})
}

func (v *visitor) handleTypeSpec(decl *ast.GenDecl, spec *ast.TypeSpec) {
	if spec.Name == nil {
		return
	}
	id := uuid.NewString()
	pos := v.fset.Position(spec.Pos())
	end := v.fset.Position(spec.End())

	kind := typeKindOf(spec.Type)
	node := model.Node{
		ID:   id,
		Kind: model.KindTypeDecl,
		Properties: map[string]any{
			"name":        spec.Name.Name,
			"kind":        kind,
			"source":      v.sourceOf(spec),
			"file_path":   v.filePath,
			"start_line":  int64(pos.Line),
			"end_line":    int64(end.Line),
			"doc_comment": docText(decl.Doc),
			"exported":    spec.Name.IsExported(),
		},
	}
	v.state.nodes = append(v.state.nodes, node)
	v.state.typesByName[spec.Name.Name] = id

	// CONTAINS: File → TypeDecl
	v.state.edges = append(v.state.edges, model.Edge{
		Kind:       model.EdgeContains,
		SourceID:   v.fileID,
		SourceKind: model.KindFile,
		TargetID:   id,
		TargetKind: model.KindTypeDecl,
	})

	// For struct types, emit Field nodes and CONTAINS edges; collect embeddings.
	if st, ok := spec.Type.(*ast.StructType); ok {
		v.handleStructFields(id, st)
	}
}

func (v *visitor) handleStructFields(typeDeclID string, st *ast.StructType) {
	if st.Fields == nil {
		return
	}
	for _, field := range st.Fields.List {
		typeName := formatExpr(v.fset, field.Type)
		tag := ""
		if field.Tag != nil {
			tag = strings.Trim(field.Tag.Value, "`")
		}

		if len(field.Names) == 0 {
			// Embedded field: no explicit name.
			embeddedName := extractEmbeddedTypeName(field.Type)
			v.state.unresolvedEmbeds = append(v.state.unresolvedEmbeds, unresolvedEmbed{
				typeDeclID: typeDeclID,
				typeName:   embeddedName,
			})
			// Still create a Field node for the embedded type.
			v.addField(typeDeclID, embeddedName, typeName, tag, field)
			continue
		}
		for _, name := range field.Names {
			v.addField(typeDeclID, name.Name, typeName, tag, field)
		}
	}
}

func (v *visitor) addField(typeDeclID, name, typeName, tag string, field *ast.Field) {
	id := uuid.NewString()
	pos := v.fset.Position(field.Pos())
	end := v.fset.Position(field.End())

	exported := len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
	node := model.Node{
		ID:   id,
		Kind: model.KindField,
		Properties: map[string]any{
			"name":       name,
			"type_name":  typeName,
			"tag":        tag,
			"source":     v.sourceOf(field),
			"file_path":  v.filePath,
			"start_line": int64(pos.Line),
			"end_line":   int64(end.Line),
			"exported":   exported,
		},
	}
	v.state.nodes = append(v.state.nodes, node)

	// CONTAINS: TypeDecl → Field
	v.state.edges = append(v.state.edges, model.Edge{
		Kind:       model.EdgeContains,
		SourceID:   typeDeclID,
		SourceKind: model.KindTypeDecl,
		TargetID:   id,
		TargetKind: model.KindField,
	})
}

func (v *visitor) handleValueSpec(decl *ast.GenDecl, spec *ast.ValueSpec) {
	kind := "var"
	if decl.Tok == token.CONST {
		kind = "const"
	}

	typeName := ""
	if spec.Type != nil {
		typeName = formatExpr(v.fset, spec.Type)
	}

	pos := v.fset.Position(spec.Pos())
	end := v.fset.Position(spec.End())

	for _, name := range spec.Names {
		id := uuid.NewString()
		node := model.Node{
			ID:   id,
			Kind: model.KindVariable,
			Properties: map[string]any{
				"name":        name.Name,
				"kind":        kind,
				"type_name":   typeName,
				"source":      v.sourceOf(spec),
				"file_path":   v.filePath,
				"start_line":  int64(pos.Line),
				"end_line":    int64(end.Line),
				"doc_comment": docText(decl.Doc),
				"exported":    name.IsExported(),
			},
		}
		v.state.nodes = append(v.state.nodes, node)

		// CONTAINS: File → Variable
		v.state.edges = append(v.state.edges, model.Edge{
			Kind:       model.EdgeContains,
			SourceID:   v.fileID,
			SourceKind: model.KindFile,
			TargetID:   id,
			TargetKind: model.KindVariable,
		})
	}
}

// sourceOf extracts the source text for an AST node using the file's bytes.
func (v *visitor) sourceOf(node ast.Node) string {
	if node == nil || len(v.src) == 0 {
		return ""
	}
	start := v.fset.Position(node.Pos()).Offset
	end := v.fset.Position(node.End()).Offset
	if start < 0 || end > len(v.src) || start >= end {
		return ""
	}
	return string(v.src[start:end])
}

// resolveDeferred processes receiver and embed references after all TypeDecls
// in the package have been collected.
func (s *parseState) resolveDeferred() {
	for _, ur := range s.unresolvedReceivers {
		typeID, ok := s.typesByName[ur.typeName]
		if !ok {
			continue
		}
		s.edges = append(s.edges, model.Edge{
			Kind:       model.EdgeHasReceiver,
			SourceID:   ur.funcID,
			SourceKind: model.KindFunction,
			TargetID:   typeID,
			TargetKind: model.KindTypeDecl,
		})
	}
	for _, ue := range s.unresolvedEmbeds {
		typeID, ok := s.typesByName[ue.typeName]
		if !ok {
			continue
		}
		s.edges = append(s.edges, model.Edge{
			Kind:       model.EdgeEmbeds,
			SourceID:   ue.typeDeclID,
			SourceKind: model.KindTypeDecl,
			TargetID:   typeID,
			TargetKind: model.KindTypeDecl,
		})
	}
}

// ---- helper functions ----

// extractReceiverTypeName returns the base type name from a receiver FieldList.
func extractReceiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	return baseTypeName(recv.List[0].Type)
}

// baseTypeName extracts the identifier from a type expression, stripping
// pointer and generic index wrappers.
func baseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return baseTypeName(t.X)
	case *ast.IndexListExpr:
		return baseTypeName(t.X)
	}
	return ""
}

// extractEmbeddedTypeName returns the type name for an embedded (anonymous) field.
func extractEmbeddedTypeName(expr ast.Expr) string {
	return baseTypeName(expr)
}

// typeKindOf returns a string describing the kind of type (struct, interface, etc.).
func typeKindOf(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface"
	case *ast.MapType:
		return "map"
	case *ast.ArrayType:
		return "array"
	case *ast.ChanType:
		return "chan"
	case *ast.FuncType:
		return "func"
	default:
		return "alias"
	}
}

// formatFuncType formats a FuncType as "(params) results".
func formatFuncType(fset *token.FileSet, ft *ast.FuncType) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, ft); err != nil {
		return ""
	}
	// format.Node on a FuncType produces "func(...) ..." – strip the leading "func".
	s := buf.String()
	s = strings.TrimPrefix(s, "func")
	return s
}

// formatExpr formats any expression as a string.
func formatExpr(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

// docText extracts the text from a doc comment group, or returns "".
func docText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return strings.TrimSpace(cg.Text())
}
