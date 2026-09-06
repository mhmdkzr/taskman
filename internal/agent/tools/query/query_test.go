package query

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mhmdkzr/loop/internal/store"
)

func TestToolQueriesReadOnlyStore(t *testing.T) {
	st, err := store.Open(t.Context(), filepath.Join(t.TempDir(), "query.sqlite"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer st.Close()
	if _, err := st.RW().Exec(`CREATE TABLE values_table (value TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := st.RW().Exec(`INSERT INTO values_table(value) VALUES ('safe')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	out, err := Execute(context.Background(), st, Input{Query: "SELECT value FROM values_table"})
	if err != nil {
		t.Fatalf("execute query: %v", err)
	}
	if !strings.Contains(out.Table, "safe") {
		t.Fatalf("output = %q, want safe value", out.Table)
	}
	if _, err := Execute(context.Background(), st, Input{Query: "DELETE FROM values_table"}); err == nil {
		t.Fatal("write query succeeded")
	}
}

func TestToolRejectsMultipleStatements(t *testing.T) {
	if err := checkReadOnlyQuery("SELECT 1; SELECT 2"); err == nil {
		t.Fatal("multiple statements were accepted")
	}
}
