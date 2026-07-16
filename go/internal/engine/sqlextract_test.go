package engine

import "testing"

func TestExtractSQL(t *testing.T) {
	sql := ExtractSQL("```sql\nSELECT 1;\n```")
	if sql != "SELECT 1;" {
		t.Fatalf("got %q", sql)
	}
}

func TestIsIntermediateSQL(t *testing.T) {
	if !IsIntermediateSQL("-- intermediate_sql\nSELECT 1") {
		t.Fatal("expected intermediate")
	}
}
