package plugins

import (
	"fmt"
	"reflect"
)

// Merge deduplicates identical definitions, and refuses ambiguous technology or
// manifest identities. Source ordering never chooses an implementation.
func Merge(sets ...Set) (Set, error) {
	result := Set{byTechnology: map[string]Manifest{}}
	ids := map[string]string{}
	for _, set := range sets {
		for technology, m := range set.byTechnology {
			if previous, ok := result.byTechnology[technology]; ok {
				if reflect.DeepEqual(previous, m) {
					continue
				}
				return Set{}, fmt.Errorf("conflicting generators for %q (%s@%s and %s@%s); select one managed version or resolve the local directory conflict", technology, previous.ID, previous.Version, m.ID, m.Version)
			}
			if existing, ok := ids[m.ID]; ok && existing != technology {
				return Set{}, fmt.Errorf("conflicting plugin ID %q", m.ID)
			}
			result.byTechnology[technology] = m
			ids[m.ID] = technology
		}
	}
	return result, nil
}
func FromManifest(manifest Manifest) Set {
	return Set{byTechnology: map[string]Manifest{manifest.TechnologyID: manifest}}
}
