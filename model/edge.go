package model

// EdgeKind identifies the type of a relationship in the property graph.
type EdgeKind string

const (
	EdgeContains    EdgeKind = "CONTAINS"
	EdgeImports     EdgeKind = "IMPORTS"
	EdgeCalls       EdgeKind = "CALLS"
	EdgeRefersTo    EdgeKind = "REFERS_TO"
	EdgeImplements  EdgeKind = "IMPLEMENTS"
	EdgeEmbeds      EdgeKind = "EMBEDS"
	EdgeHasReceiver EdgeKind = "HAS_RECEIVER"
	EdgeResolvesTo  EdgeKind = "RESOLVES_TO"
)

// Edge is a directed relationship between two nodes in the property graph.
type Edge struct {
	Kind       EdgeKind
	SourceID   string
	SourceKind NodeKind
	TargetID   string
	TargetKind NodeKind
	Properties map[string]any
}
