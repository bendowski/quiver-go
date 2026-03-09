package schema_test

import (
	"testing"

	lbug "github.com/LadybugDB/go-ladybug"
	"github.com/bendowski/quiver/schema"
)

func TestInitSchema(t *testing.T) {
	db, err := lbug.OpenInMemoryDatabase(lbug.DefaultSystemConfig())
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer db.Close()

	conn, err := lbug.OpenConnection(db)
	if err != nil {
		t.Fatalf("open connection: %v", err)
	}
	defer conn.Close()

	execFn := func(stmt string) error {
		r, err := conn.Query(stmt)
		if err != nil {
			return err
		}
		r.Close()
		return nil
	}

	if err := schema.InitSchema(execFn); err != nil {
		t.Fatalf("InitSchema: %v", err)
	}

	// Verify at least one known node table exists by querying it.
	r, err := conn.Query("MATCH (n:Package) RETURN n")
	if err != nil {
		t.Fatalf("query Package after schema init: %v", err)
	}
	r.Close()
}
