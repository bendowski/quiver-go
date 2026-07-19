// Package cypher provides Cypher query string helpers and a CypherQuerier
// implementation backed by a LadyBug connection.
package cypher

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/bendowski/quiver/model"
)

// propClauses stores each property in params under a prefix-namespaced key
// and returns the matching "name: $prefix_name" clause fragments. Keys are
// emitted in sorted order so generated statements are deterministic.
func propClauses(prefix string, props map[string]any, params map[string]any) []string {
	parts := make([]string, 0, len(props))
	for _, k := range slices.Sorted(maps.Keys(props)) {
		paramKey := prefix + k
		params[paramKey] = props[k]
		parts = append(parts, k+": $"+paramKey)
	}
	return parts
}

// BuildCreateNode returns a Cypher CREATE statement and its parameter map for
// inserting a node of the given kind. The node id is passed separately and is
// always included as property "id".
//
// Parameter names are prefixed with "p_" to avoid conflicts with Cypher
// reserved words.
func BuildCreateNode(kind model.NodeKind, id string, props map[string]any) (string, map[string]any) {
	params := make(map[string]any, len(props)+1)

	// id is always first.
	params["p_id"] = id
	parts := append([]string{"id: $p_id"}, propClauses("p_", props, params)...)

	stmt := fmt.Sprintf("CREATE (n:%s {%s})", kind, strings.Join(parts, ", "))
	return stmt, params
}

// BuildCreateEdge returns a Cypher MATCH…CREATE statement and its parameter
// map for inserting a relationship. Extra edge properties are supported.
//
// Parameter names: p_src, p_dst for node IDs; ep_<key> for edge properties.
func BuildCreateEdge(
	srcKind model.NodeKind, srcID string,
	dstKind model.NodeKind, dstID string,
	edgeKind model.EdgeKind,
	props map[string]any,
) (string, map[string]any) {
	params := map[string]any{
		"p_src": srcID,
		"p_dst": dstID,
	}

	var relProps string
	if len(props) > 0 {
		relProps = " {" + strings.Join(propClauses("ep_", props, params), ", ") + "}"
	}

	stmt := fmt.Sprintf(
		"MATCH (a:%s {id: $p_src}), (b:%s {id: $p_dst}) CREATE (a)-[:%s%s]->(b)",
		srcKind, dstKind, edgeKind, relProps,
	)
	return stmt, params
}

// BuildMatchByKind returns a Cypher MATCH statement that returns all nodes of
// the given kind.
func BuildMatchByKind(kind model.NodeKind) string {
	return fmt.Sprintf("MATCH (n:%s) RETURN n", kind)
}

// propNamePattern matches plain identifiers — the only shape a
// schema-defined property name can have.
var propNamePattern = regexp.MustCompile(`^[A-Za-z_]\w*$`)

// BuildMatchByProperty returns a Cypher MATCH statement and its parameter map
// to find nodes whose property prop equals val.
//
// prop is interpolated directly into the query string and must be a trusted,
// schema-defined property name; BuildMatchByProperty panics if prop is not a
// plain identifier, so a would-be injection fails loudly instead of reaching
// the database.
func BuildMatchByProperty(kind model.NodeKind, prop string, val any) (string, map[string]any) {
	if !propNamePattern.MatchString(prop) {
		panic(fmt.Sprintf("cypher: invalid property name %q", prop))
	}
	params := map[string]any{"p_val": val}
	stmt := fmt.Sprintf("MATCH (n:%s) WHERE n.%s = $p_val RETURN n", kind, prop)
	return stmt, params
}

// BuildMatchEdgesFrom returns a Cypher MATCH statement and its parameter map
// that returns all outgoing relationships from a node identified by nodeID in
// table nodeKind. Each result row contains columns "r" (relationship) and "b"
// (target node).
func BuildMatchEdgesFrom(nodeKind model.NodeKind, nodeID string) (string, map[string]any) {
	params := map[string]any{"p_id": nodeID}
	stmt := fmt.Sprintf("MATCH (a:%s {id: $p_id})-[r]->(b) RETURN r, b", nodeKind)
	return stmt, params
}
