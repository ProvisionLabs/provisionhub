package catalog

import (
	"fmt"
	"slices"
)

type Category string

const (
	Backend  Category = "backend"
	Frontend Category = "frontend"
	Database Category = "database"
)

type Technology struct {
	ID               string
	Name             string
	Category         Category
	Description      string
	IncompatibleWith []string // Optional explicit technology IDs; empty for built-ins.
}

// Catalog is an immutable snapshot of available technology definitions.
type Catalog struct {
	entries []Technology
	byID    map[string]Technology
}

func New(entries []Technology) (Catalog, error) {
	copyOfEntries := slices.Clone(entries)
	byID := make(map[string]Technology, len(entries))
	for i, entry := range copyOfEntries {
		if entry.ID == "" || entry.Name == "" || entry.Description == "" {
			return Catalog{}, fmt.Errorf("technology %q must have an ID, name, and description", entry.ID)
		}
		if entry.Category != Backend && entry.Category != Frontend && entry.Category != Database {
			return Catalog{}, fmt.Errorf("technology %q has unsupported category %q", entry.ID, entry.Category)
		}
		if _, exists := byID[entry.ID]; exists {
			return Catalog{}, fmt.Errorf("duplicate technology ID %q", entry.ID)
		}
		entry.IncompatibleWith = slices.Clone(entry.IncompatibleWith)
		copyOfEntries[i] = entry
		byID[entry.ID] = entry
	}
	slices.SortFunc(copyOfEntries, func(a, b Technology) int {
		if a.Category != b.Category {
			if a.Category < b.Category {
				return -1
			}
			return 1
		}
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return Catalog{entries: copyOfEntries, byID: byID}, nil
}

func Builtin() Catalog {
	entries := []Technology{
		{ID: "nestjs", Name: "NestJS", Category: Backend, Description: "Node.js backend framework"},
		{ID: "spring-boot", Name: "Spring Boot", Category: Backend, Description: "Java backend framework"},
		{ID: "go", Name: "Go", Category: Backend, Description: "Go backend application"},
		{ID: "nextjs", Name: "Next.js", Category: Frontend, Description: "React application framework"},
		{ID: "react", Name: "React", Category: Frontend, Description: "React frontend application"},
		{ID: "postgresql", Name: "PostgreSQL", Category: Database, Description: "Relational database"},
		{ID: "mysql", Name: "MySQL", Category: Database, Description: "Relational database"},
	}
	catalog, err := New(entries)
	if err != nil {
		panic(err) // Built-in definitions are fixed and validated during development.
	}
	return catalog
}

func (c Catalog) List(category Category) []Technology {
	var result []Technology
	for _, entry := range c.entries {
		if entry.Category == category {
			entry.IncompatibleWith = slices.Clone(entry.IncompatibleWith)
			result = append(result, entry)
		}
	}
	return result
}

func (c Catalog) Lookup(id string) (Technology, bool) {
	entry, ok := c.byID[id]
	entry.IncompatibleWith = slices.Clone(entry.IncompatibleWith)
	return entry, ok
}

func (c Catalog) ValidateSelection(category Category, id string) (Technology, error) {
	entry, ok := c.Lookup(id)
	if !ok {
		return Technology{}, fmt.Errorf("unknown %s technology %q", category, id)
	}
	if entry.Category != category {
		return Technology{}, fmt.Errorf("technology %q belongs to %s, not %s", id, entry.Category, category)
	}
	return entry, nil
}

// ValidateCombination rejects only explicitly declared incompatibilities.
func (c Catalog) ValidateCombination(ids ...string) error {
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		if _, ok := c.byID[id]; !ok {
			return fmt.Errorf("unknown technology %q", id)
		}
		selected[id] = true
	}
	for _, id := range ids {
		for _, incompatible := range c.byID[id].IncompatibleWith {
			if selected[incompatible] {
				return fmt.Errorf("technologies %q and %q are incompatible", id, incompatible)
			}
		}
	}
	return nil
}
