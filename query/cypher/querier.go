package cypher

import (
	"context"
	"fmt"

	lbug "github.com/LadybugDB/go-ladybug"
	"github.com/bendowski/quiver/query"
)

// CypherQuerier executes raw Cypher queries against a LadyBug connection and
// returns structured query.Result values.
type CypherQuerier struct {
	conn *lbug.Connection
}

// New creates a new CypherQuerier using the provided connection.
func New(conn *lbug.Connection) *CypherQuerier {
	return &CypherQuerier{conn: conn}
}

// Query executes a Cypher string with no parameters.
func (q *CypherQuerier) Query(_ context.Context, cypher string) (*query.Result, error) {
	qr, err := q.conn.Query(cypher)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer qr.Close()
	return collectResult(qr)
}

// QueryWithParams executes a Cypher string with named parameters.
func (q *CypherQuerier) QueryWithParams(_ context.Context, cypher string, params map[string]any) (*query.Result, error) {
	ps, err := q.conn.Prepare(cypher)
	if err != nil {
		return nil, fmt.Errorf("prepare: %w", err)
	}
	defer ps.Close()
	qr, err := q.conn.Execute(ps, params)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}
	defer qr.Close()
	return collectResult(qr)
}

// collectResult converts a LadyBug QueryResult into a query.Result.
func collectResult(qr *lbug.QueryResult) (*query.Result, error) {
	columns := qr.GetColumnNames()
	result := &query.Result{Columns: columns}

	for qr.HasNext() {
		tuple, err := qr.Next()
		if err != nil {
			return nil, fmt.Errorf("next: %w", err)
		}
		m, err := tuple.GetAsMap()
		if err != nil {
			return nil, fmt.Errorf("getAsMap: %w", err)
		}
		result.Rows = append(result.Rows, query.Row(m))
	}
	return result, nil
}
