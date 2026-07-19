package model

// NodeKind identifies the type of a graph node.
type NodeKind string

const (
	KindPackage  NodeKind = "Package"
	KindFile     NodeKind = "File"
	KindImport   NodeKind = "Import"
	KindFunction NodeKind = "Function"
	KindTypeDecl NodeKind = "TypeDecl"
	KindVariable NodeKind = "Variable"
	KindField    NodeKind = "Field"
)

// AllNodeKinds lists every NodeKind in dependency order (no forward refs).
var AllNodeKinds = []NodeKind{
	KindPackage,
	KindFile,
	KindImport,
	KindFunction,
	KindTypeDecl,
	KindVariable,
	KindField,
}

// Node is a vertex in the source-code property graph.
type Node struct {
	ID         string // UUID assigned client-side
	Kind       NodeKind
	Properties map[string]any
}
