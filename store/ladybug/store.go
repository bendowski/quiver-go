// Package ladybug provides a Store implementation backed by LadyBug
// (an embedded columnar graph database with Cypher support).
package ladybug

import (
	"context"
	"fmt"

	lbug "github.com/LadybugDB/go-ladybug"
	"github.com/bendowski/quiver/model"
	"github.com/bendowski/quiver/query/cypher"
	"github.com/bendowski/quiver/schema"
)

// Store implements store.Store on top of a LadyBug database.
type Store struct {
	db   *lbug.Database
	conn *lbug.Connection
}

// Open opens (or creates) a LadyBug database at path. When path is empty an
// in-memory database is used. InitSchema is called automatically on the new
// connection.
func Open(path string) (*Store, error) {
	config := lbug.DefaultSystemConfig()
	var db *lbug.Database
	var err error
	if path == "" {
		db, err = lbug.OpenInMemoryDatabase(config)
	} else {
		db, err = lbug.OpenDatabase(path, config)
	}
	if err != nil {
		return nil, fmt.Errorf("ladybug open db: %w", err)
	}

	conn, err := lbug.OpenConnection(db)
	if err != nil {
		return nil, fmt.Errorf("ladybug open connection: %w", err)
	}

	s := &Store{db: db, conn: conn}
	if err := schema.InitSchema(s.exec); err != nil {
		conn.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}
	return s, nil
}

// Connection returns the underlying LadyBug connection. This is intentionally
// not part of the store.Store interface; callers that need raw Cypher access
// (e.g. the CLI query command) retrieve it here and pass it to cypher.New.
func (s *Store) Connection() *lbug.Connection { return s.conn }

// exec runs a Cypher statement with no parameters and discards the result.
func (s *Store) exec(stmt string) error {
	r, err := s.conn.Query(stmt)
	if err != nil {
		return err
	}
	r.Close()
	return nil
}

// execParams prepares and executes a parameterised Cypher statement.
func (s *Store) execParams(stmt string, params map[string]any) error {
	ps, err := s.conn.Prepare(stmt)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer ps.Close()
	r, err := s.conn.Execute(ps, params)
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	r.Close()
	return nil
}

// AddNode persists a node. The node's ID must already be set.
func (s *Store) AddNode(_ context.Context, node model.Node) error {
	stmt, params := cypher.BuildCreateNode(node.Kind, node.ID, node.Properties)
	if err := s.execParams(stmt, params); err != nil {
		return fmt.Errorf("AddNode %s %s: %w", node.Kind, node.ID, err)
	}
	return nil
}

// AddEdge persists a directed relationship between two existing nodes.
func (s *Store) AddEdge(_ context.Context, edge model.Edge) error {
	stmt, params := cypher.BuildCreateEdge(
		edge.SourceKind, edge.SourceID,
		edge.TargetKind, edge.TargetID,
		edge.Kind, edge.Properties,
	)
	if err := s.execParams(stmt, params); err != nil {
		return fmt.Errorf("AddEdge %s→%s: %w", edge.SourceID, edge.TargetID, err)
	}
	return nil
}

// GetNodesByKind returns all nodes of the given kind.
func (s *Store) GetNodesByKind(_ context.Context, kind model.NodeKind) ([]model.Node, error) {
	stmt := cypher.BuildMatchByKind(kind)
	qr, err := s.conn.Query(stmt)
	if err != nil {
		return nil, fmt.Errorf("GetNodesByKind %s: %w", kind, err)
	}
	defer qr.Close()
	return collectNodes(qr, kind)
}

// GetEdgesFrom returns all outgoing edges from the specified node.
func (s *Store) GetEdgesFrom(_ context.Context, nodeID string, nodeKind model.NodeKind) ([]model.Edge, error) {
	stmt, params := cypher.BuildMatchEdgesFrom(nodeKind, nodeID)
	ps, err := s.conn.Prepare(stmt)
	if err != nil {
		return nil, fmt.Errorf("GetEdgesFrom prepare: %w", err)
	}
	defer ps.Close()
	qr, err := s.conn.Execute(ps, params)
	if err != nil {
		return nil, fmt.Errorf("GetEdgesFrom execute: %w", err)
	}
	defer qr.Close()
	return collectEdges(qr, nodeID, nodeKind)
}

// FindNodeByProperty returns nodes of kind where property prop equals val.
func (s *Store) FindNodeByProperty(_ context.Context, kind model.NodeKind, prop string, val any) ([]model.Node, error) {
	stmt, params := cypher.BuildMatchByProperty(kind, prop, val)
	ps, err := s.conn.Prepare(stmt)
	if err != nil {
		return nil, fmt.Errorf("FindNodeByProperty prepare: %w", err)
	}
	defer ps.Close()
	qr, err := s.conn.Execute(ps, params)
	if err != nil {
		return nil, fmt.Errorf("FindNodeByProperty execute: %w", err)
	}
	defer qr.Close()
	return collectNodes(qr, kind)
}

// Clear deletes all nodes (and their edges) from every table.
func (s *Store) Clear(_ context.Context) error {
	for _, kind := range model.AllNodeKinds {
		stmt := fmt.Sprintf("MATCH (n:%s) DETACH DELETE n", kind)
		if err := s.exec(stmt); err != nil {
			return fmt.Errorf("Clear %s: %w", kind, err)
		}
	}
	return nil
}

// Close closes the connection and database.
func (s *Store) Close() error {
	s.conn.Close()
	s.db.Close()
	return nil
}

// collectNodes scans a QueryResult, expecting each row to have a single column
// "n" containing an lbug.Node, and converts them to model.Node values.
func collectNodes(qr *lbug.QueryResult, kind model.NodeKind) ([]model.Node, error) {
	var nodes []model.Node
	for qr.HasNext() {
		tuple, err := qr.Next()
		if err != nil {
			return nil, fmt.Errorf("next: %w", err)
		}
		m, err := tuple.GetAsMap()
		if err != nil {
			return nil, fmt.Errorf("getAsMap: %w", err)
		}
		raw, ok := m["n"]
		if !ok {
			return nil, fmt.Errorf("column 'n' missing")
		}
		lbNode, ok := raw.(lbug.Node)
		if !ok {
			return nil, fmt.Errorf("unexpected node type: %T", raw)
		}
		// Copy properties; id lives in Properties as well.
		props := make(map[string]any, len(lbNode.Properties))
		for k, v := range lbNode.Properties {
			props[k] = v
		}
		id, _ := props["id"].(string)
		nodes = append(nodes, model.Node{
			ID:         id,
			Kind:       kind,
			Properties: props,
		})
	}
	return nodes, nil
}

// collectEdges scans a QueryResult with columns "r" (lbug.Relationship) and
// "b" (lbug.Node) and builds model.Edge values.
func collectEdges(qr *lbug.QueryResult, srcID string, srcKind model.NodeKind) ([]model.Edge, error) {
	var edges []model.Edge
	for qr.HasNext() {
		tuple, err := qr.Next()
		if err != nil {
			return nil, fmt.Errorf("next: %w", err)
		}
		m, err := tuple.GetAsMap()
		if err != nil {
			return nil, fmt.Errorf("getAsMap: %w", err)
		}

		rawRel, ok := m["r"]
		if !ok {
			return nil, fmt.Errorf("column 'r' missing")
		}
		rel, ok := rawRel.(lbug.Relationship)
		if !ok {
			return nil, fmt.Errorf("unexpected relationship type: %T", rawRel)
		}

		rawNode, ok := m["b"]
		if !ok {
			return nil, fmt.Errorf("column 'b' missing")
		}
		dstNode, ok := rawNode.(lbug.Node)
		if !ok {
			return nil, fmt.Errorf("unexpected node type: %T", rawNode)
		}

		dstID, _ := dstNode.Properties["id"].(string)
		relProps := make(map[string]any, len(rel.Properties))
		for k, v := range rel.Properties {
			relProps[k] = v
		}

		edges = append(edges, model.Edge{
			Kind:       model.EdgeKind(rel.Label),
			SourceID:   srcID,
			SourceKind: srcKind,
			TargetID:   dstID,
			TargetKind: model.NodeKind(dstNode.Label),
			Properties: relProps,
		})
	}
	return edges, nil
}
