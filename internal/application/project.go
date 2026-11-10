package application

import (
	"fmt"
	"maps"
	"os"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

type ProjectService struct {
	catalog         catalog.Catalog
	pluginDirectory string
	pluginVersions  map[string]string
	pluginManager   *PluginManager
	localOnly       bool
}

func (s ProjectService) manager() PluginManager {
	if s.pluginManager != nil {
		return *s.pluginManager
	}
	return PluginManager{}
}

func NewProjectService() ProjectService {
	return ProjectService{catalog: catalog.Builtin()}
}

// NewProjectServiceWithPlugins uses explicitly supplied local definitions
// without opening SQLite. A selected managed version opts into managed lookup.
func NewProjectServiceWithPlugins(directory string) (ProjectService, error) {
	s := NewProjectService()
	resolved, err := pluginDirectory(directory)
	if err != nil {
		return ProjectService{}, err
	}
	if _, err := plugins.LoadDirectory(resolved, s.catalog); err != nil {
		return ProjectService{}, err
	}
	s.pluginDirectory = directory
	s.localOnly = directory != "" || os.Getenv("PROVISIONHUB_PLUGINS") != ""
	return s, nil
}

func (s ProjectService) Technologies(category catalog.Category) []catalog.Technology {
	return s.catalog.List(category)
}

func (s ProjectService) TechnologyName(id string) string {
	if id == "" || id == "none" {
		return "None"
	}
	if technology, ok := s.catalog.Lookup(id); ok {
		return technology.Name
	}
	return id
}

func (s ProjectService) ValidateProjectInfo(name, destination string) (string, error) {
	if err := project.ValidateName(name); err != nil {
		return "", err
	}
	return project.ValidateDestination(destination)
}

// ValidateComponentsInput constructs a project specification in memory.
func (s ProjectService) ValidateComponentsInput(name, destination string, components []project.Component) (project.Specification, error) {
	if _, err := s.ValidateProjectInfo(name, destination); err != nil {
		return project.Specification{}, err
	}
	spec := project.Specification{APIVersion: project.APIVersion, Kind: project.Kind,
		Metadata: project.Metadata{Name: name}, Spec: project.Spec{Components: append([]project.Component{}, components...)}}
	for i := range spec.Spec.Components {
		spec.Spec.Components[i].Configuration = maps.Clone(spec.Spec.Components[i].Configuration)
	}
	if err := spec.Validate(); err != nil {
		return project.Specification{}, err
	}
	for _, component := range spec.CanonicalComponents() {
		if _, err := s.catalog.ValidateSelection(catalog.Category(component.Type), component.Technology); err != nil {
			return project.Specification{}, fmt.Errorf("component %q: %w", component.Name, err)
		}
	}
	if err := s.catalog.ValidateCombination(selectionIDs(spec)...); err != nil {
		return project.Specification{}, err
	}
	return spec, nil
}

func selectionIDs(spec project.Specification) []string {
	components := spec.CanonicalComponents()
	ids := make([]string, 0, len(components))
	for _, component := range components {
		ids = append(ids, component.Technology)
	}
	return ids
}
