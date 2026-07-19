package schema_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/bendowski/quiver/schema"
)

// These tests exercise InitSchema against a fake ExecFunc so they need no
// database. Execution of the DDL against a real LadyBug instance is covered
// by the store/ladybug tests, whose Open("") calls InitSchema on every run.

func TestInitSchema_allStatementsExecuted(t *testing.T) {
	var stmts []string
	err := schema.InitSchema(func(stmt string) error {
		stmts = append(stmts, stmt)
		return nil
	})
	if err != nil {
		t.Fatalf("InitSchema: %v", err)
	}
	if len(stmts) != 15 { // 7 node tables + 8 rel tables
		t.Fatalf("want 15 statements, got %d", len(stmts))
	}
	// Node tables must come before rel tables (rel DDL references node tables).
	if !strings.HasPrefix(stmts[0], "CREATE NODE TABLE") {
		t.Errorf("first stmt is not a node table: %s", stmts[0])
	}
	if !strings.HasPrefix(stmts[14], "CREATE REL TABLE") {
		t.Errorf("last stmt is not a rel table: %s", stmts[14])
	}
}

func TestInitSchema_alreadyExistsIsIgnored(t *testing.T) {
	err := schema.InitSchema(func(stmt string) error {
		return errors.New(`Binder exception: Foo already exists in catalog.`)
	})
	if err != nil {
		t.Fatalf("already-exists errors should be skipped, got: %v", err)
	}
}

func TestInitSchema_otherErrorsPropagate(t *testing.T) {
	boom := errors.New("disk exploded")
	err := schema.InitSchema(func(stmt string) error { return boom })
	if !errors.Is(err, boom) {
		t.Fatalf("want propagated error, got: %v", err)
	}
}
