package registry

import "fmt"

// RetainHistory adds current build releases without deleting or replacing known
// immutable versions. Shared metadata may evolve, but identity must stay stable.
func RetainHistory(current, previous Index) (Index, error) {
	if err := current.Validate(); err != nil {
		return Index{}, err
	}
	if err := previous.Validate(); err != nil {
		return Index{}, err
	}
	result := Index{SchemaVersion: 1, Plugins: append([]Plugin{}, current.Plugins...)}
	for _, old := range previous.Plugins {
		found := false
		for i, p := range result.Plugins {
			if p.ID != old.ID {
				continue
			}
			found = true
			if p.ManifestID != old.ManifestID || p.Publisher != old.Publisher || p.TechnologyID != old.TechnologyID || p.Category != old.Category {
				return Index{}, fmt.Errorf("plugin %q changes immutable identity metadata", p.ID)
			}
			versions := append([]Release{}, p.Versions...)
			for _, release := range old.Versions {
				exists := false
				for _, built := range p.Versions {
					if built.Version == release.Version {
						exists = true
						if built != release {
							return Index{}, fmt.Errorf("refusing to replace immutable release %s@%s", p.ID, release.Version)
						}
					}
				}
				if !exists {
					versions = append(versions, release)
				}
			}
			result.Plugins[i].Versions = versions
		}
		if !found {
			result.Plugins = append(result.Plugins, old)
		}
	}
	return result, result.Validate()
}
