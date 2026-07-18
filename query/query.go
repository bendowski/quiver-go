// Package query defines the Querier interface for executing raw Cypher queries.
package query

import "context"

// Row is a single result row, keyed by column name.
type Row map[string]any

// Result holds the full result of a Cypher query.
type Result struct {
	Columns []string
	Rows    []Row
}

// Querier executes raw Cypher queries against the graph database.
type Querier interface {
	// Query executes a Cypher string with no parameters.
	Query(ctx context.Context, cypher string) (*Result, error)

	// QueryWithParams executes a Cypher string with named parameters.
	QueryWithParams(ctx context.Context, cypher string, params map[string]any) (*Result, error)
}
