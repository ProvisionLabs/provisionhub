package catalog

import (
	"reflect"
	"testing"
)

func TestBuiltinCatalog(t *testing.T) {
	c := Builtin()
	for _, test := range []struct {
		category Category
		ids      []string
	}{
		{Backend, []string{"go", "nestjs", "spring-boot"}},
		{Frontend, []string{"nextjs", "react"}},
		{Database, []string{"mysql", "postgresql"}},
	} {
		var got []string
		for _, entry := range c.List(test.category) {
			got = append(got, entry.ID)
			found, ok := c.Lookup(entry.ID)
			if !ok || !reflect.DeepEqual(found, entry) {
				t.Errorf("lookup of %q returned %+v, %v", entry.ID, found, ok)
			}
		}
		if !reflect.DeepEqual(got, test.ids) {
			t.Errorf("%s IDs = %v, want %v", test.category, got, test.ids)
		}
	}
	if _, ok := c.Lookup("absent"); ok {
		t.Fatal("unknown technology was found")
	}
	if len(c.List("unknown")) != 0 {
		t.Fatal("unknown category should be empty")
	}
}

func TestValidateSelection(t *testing.T) {
	c := Builtin()
	if _, err := c.ValidateSelection(Backend, "nestjs"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ValidateSelection(Backend, "nextjs"); err == nil {
		t.Fatal("accepted a frontend as a backend")
	}
	if _, err := c.ValidateSelection(Database, "unknown"); err == nil {
		t.Fatal("accepted an unknown technology")
	}
}

func TestNewRejectsDuplicateIDsAndCopiesInput(t *testing.T) {
	entries := []Technology{{ID: "one", Name: "One", Category: Backend, Description: "One backend"}}
	c, err := New(entries)
	if err != nil {
		t.Fatal(err)
	}
	entries[0].Name = "Changed"
	if got, _ := c.Lookup("one"); got.Name != "One" {
		t.Fatal("catalog retained a mutable input slice")
	}
	if _, err := New([]Technology{entries[0], entries[0]}); err == nil {
		t.Fatal("duplicate IDs should be rejected")
	}
}

func TestExplicitCompatibilityMetadata(t *testing.T) {
	entries := []Technology{
		{ID: "backend-a", Name: "Backend A", Category: Backend, Description: "Backend", IncompatibleWith: []string{"database-b"}},
		{ID: "database-b", Name: "Database B", Category: Database, Description: "Database"},
	}
	c, err := New(entries)
	if err != nil {
		t.Fatal(err)
	}
	entries[0].IncompatibleWith[0] = "changed"
	if err := c.ValidateCombination("backend-a", "database-b"); err == nil {
		t.Fatal("explicitly incompatible technologies were accepted")
	}
	if err := c.ValidateCombination("backend-a"); err != nil {
		t.Fatal(err)
	}
	copyOfEntry, _ := c.Lookup("backend-a")
	copyOfEntry.IncompatibleWith[0] = "changed"
	if err := c.ValidateCombination("backend-a", "database-b"); err == nil {
		t.Fatal("lookup exposed mutable catalog metadata")
	}
	if err := Builtin().ValidateCombination("nestjs", "nextjs", "postgresql"); err != nil {
		t.Fatalf("built-in combinations should be valid: %v", err)
	}
}
