// Package schema defines the LadyBug DDL for the quiver property graph and
// provides InitSchema to execute those statements against a live connection.
package schema

import (
	"slices"
	"strings"
)

// nodeTableDDL lists CREATE NODE TABLE statements in dependency order.
var nodeTableDDL = []string{
	`CREATE NODE TABLE Package(id STRING PRIMARY KEY, name STRING, import_path STRING, dir STRING)`,
	`CREATE NODE TABLE File(id STRING PRIMARY KEY, name STRING, file_path STRING, package_name STRING, source STRING)`,
	`CREATE NODE TABLE Import(id STRING PRIMARY KEY, alias STRING, path STRING, file_path STRING, start_line INT64, end_line INT64)`,
	`CREATE NODE TABLE Function(id STRING PRIMARY KEY, name STRING, signature STRING, receiver STRING, source STRING, file_path STRING, start_line INT64, end_line INT64, doc_comment STRING, exported BOOL)`,
	`CREATE NODE TABLE TypeDecl(id STRING PRIMARY KEY, name STRING, kind STRING, source STRING, file_path STRING, start_line INT64, end_line INT64, doc_comment STRING, exported BOOL)`,
	`CREATE NODE TABLE Variable(id STRING PRIMARY KEY, name STRING, kind STRING, type_name STRING, source STRING, file_path STRING, start_line INT64, end_line INT64, doc_comment STRING, exported BOOL)`,
	`CREATE NODE TABLE Field(id STRING PRIMARY KEY, name STRING, type_name STRING, tag STRING, source STRING, file_path STRING, start_line INT64, end_line INT64, exported BOOL)`,
}

// relTableDDL lists CREATE REL TABLE statements.
var relTableDDL = []string{
	`CREATE REL TABLE CONTAINS(FROM Package TO File, FROM File TO Function, FROM File TO TypeDecl, FROM File TO Variable, FROM TypeDecl TO Field)`,
	`CREATE REL TABLE IMPORTS(FROM File TO Import)`,
	`CREATE REL TABLE CALLS(FROM Function TO Function, file_path STRING, line INT64)`,
	`CREATE REL TABLE REFERS_TO(FROM Function TO TypeDecl, FROM Function TO Variable, file_path STRING, line INT64)`,
	`CREATE REL TABLE IMPLEMENTS(FROM TypeDecl TO TypeDecl)`,
	`CREATE REL TABLE EMBEDS(FROM TypeDecl TO TypeDecl)`,
	`CREATE REL TABLE HAS_RECEIVER(FROM Function TO TypeDecl)`,
	`CREATE REL TABLE RESOLVES_TO(FROM Import TO Package)`,
}

// ExecFunc executes a single DDL statement against a database connection.
type ExecFunc func(stmt string) error

// InitSchema executes all DDL statements via execFn. Errors indicating that a
// table already exists are silently ignored, so InitSchema is safe to call
// against an already-initialised database.
func InitSchema(execFn ExecFunc) error {
	for _, stmt := range slices.Concat(nodeTableDDL, relTableDDL) {
		if err := execFn(stmt); err != nil {
			if isAlreadyExistsError(err) {
				continue
			}
			return err
		}
	}
	return nil
}

// isAlreadyExistsError returns true when err is a LadyBug "already exists in
// catalog" error, which means the table was created in a previous session.
//
// go-ladybug exposes no typed errors, so matching the message text is the
// only option; the substring is pinned to upstream's wording and must be
// re-checked when bumping the go-ladybug version.
func isAlreadyExistsError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "already exists in catalog")
}
