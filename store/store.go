// Package store defines the typed CRUD interface over the property graph DB.
package store

import (
	"context"
	"io"

	"github.com/bendowski/quiver/model"
)

// Store is the typed CRUD interface for the property graph.
type Store interface {
	// AddNode persists a node. The node's ID must be set by the caller.
	AddNode(ctx context.Context, node model.Node) error

	// AddEdge persists a directed relationship between two existing nodes.
	AddEdge(ctx context.Context, edge model.Edge) error

	// GetNodesByKind returns all nodes of the given kind.
	GetNodesByKind(ctx context.Context, kind model.NodeKind) ([]model.Node, error)

	// GetEdgesFrom returns all outgoing edges from the node identified by
	// nodeID (which must be in table nodeKind).
	GetEdgesFrom(ctx context.Context, nodeID string, nodeKind model.NodeKind) ([]model.Edge, error)

	// FindNodeByProperty returns all nodes of kind whose property prop equals val.
	FindNodeByProperty(ctx context.Context, kind model.NodeKind, prop string, val any) ([]model.Node, error)

	// Clear removes all nodes and edges from the database.
	Clear(ctx context.Context) error

	io.Closer
}
