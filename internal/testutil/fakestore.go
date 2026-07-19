package testutil

import (
	"context"
	"maps"

	"github.com/bendowski/quiver/model"
	"github.com/bendowski/quiver/store"
)

// FakeStore is a pure in-memory store.Store implementation for tests that
// don't need a real LadyBug database. Mirroring the LadyBug store, each
// node's ID is also exposed as property "id", so lookups like
// FindNodeByProperty(kind, "id", id) behave the same. Result ordering is
// unspecified. FakeStore is not safe for concurrent use.
type FakeStore struct {
	nodes map[string]model.Node   // by node ID
	edges map[string][]model.Edge // outgoing edges by source node ID
}

var _ store.Store = (*FakeStore)(nil)

// NewFakeStore returns an empty FakeStore.
func NewFakeStore() *FakeStore {
	return &FakeStore{
		nodes: make(map[string]model.Node),
		edges: make(map[string][]model.Edge),
	}
}

// AddNode stores a copy of node, exposing its ID as property "id".
func (f *FakeStore) AddNode(_ context.Context, node model.Node) error {
	props := maps.Clone(node.Properties)
	if props == nil {
		props = make(map[string]any, 1)
	}
	props[model.PropID] = node.ID
	node.Properties = props
	f.nodes[node.ID] = node
	return nil
}

// AddEdge stores edge under its source node ID.
func (f *FakeStore) AddEdge(_ context.Context, edge model.Edge) error {
	f.edges[edge.SourceID] = append(f.edges[edge.SourceID], edge)
	return nil
}

// GetNodesByKind returns all nodes of the given kind.
func (f *FakeStore) GetNodesByKind(_ context.Context, kind model.NodeKind) ([]model.Node, error) {
	var out []model.Node
	for _, n := range f.nodes {
		if n.Kind == kind {
			out = append(out, n)
		}
	}
	return out, nil
}

// GetEdgesFrom returns all outgoing edges of the node with the given ID.
func (f *FakeStore) GetEdgesFrom(_ context.Context, nodeID string, _ model.NodeKind) ([]model.Edge, error) {
	return f.edges[nodeID], nil
}

// FindNodeByProperty returns all nodes of kind whose property prop equals val.
func (f *FakeStore) FindNodeByProperty(_ context.Context, kind model.NodeKind, prop string, val any) ([]model.Node, error) {
	var out []model.Node
	for _, n := range f.nodes {
		if n.Kind == kind && n.Properties[prop] == val {
			out = append(out, n)
		}
	}
	return out, nil
}

// Clear removes all nodes and edges.
func (f *FakeStore) Clear(_ context.Context) error {
	f.nodes = make(map[string]model.Node)
	f.edges = make(map[string][]model.Edge)
	return nil
}

// Close is a no-op.
func (f *FakeStore) Close() error { return nil }
