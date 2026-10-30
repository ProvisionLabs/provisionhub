package application

import (
	"fmt"

	"github.com/ProvisionLabs/provisionhub/internal/catalog"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

type CreateProjectInput struct {
	Name        string
	Destination string
	Backend     string
	Frontend    string
	Database    string
}

type CreateProjectResult struct {
	Manifest     project.Manifest
	ManifestPath string
}

type ProjectService struct {
	catalog         catalog.Catalog
	pluginDirectory string
}

func NewProjectService() ProjectService {
	return ProjectService{catalog: catalog.Builtin()}
}

func NewProjectServiceWithPlugins(directory string) (ProjectService, error) {
	s := NewProjectService()
	_, err := plugins.LoadDirectory(directory, s.catalog)
	if err != nil {
		return ProjectService{}, err
	}
	s.pluginDirectory = directory
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
	_, path, err := project.CheckDestination(destination)
	return path, err
}

// ValidateInput builds a desired-state manifest without writing it.
func (s ProjectService) ValidateInput(input CreateProjectInput) (project.Manifest, string, error) {
	path, err := s.ValidateProjectInfo(input.Name, input.Destination)
	if err != nil {
		return project.Manifest{}, "", err
	}
	if input.Backend == "" {
		return project.Manifest{}, "", fmt.Errorf("backend technology is required")
	}
	if _, err := s.catalog.ValidateSelection(catalog.Backend, input.Backend); err != nil {
		return project.Manifest{}, "", err
	}
	manifest := project.Manifest{
		APIVersion: project.APIVersion,
		Kind:       project.Kind,
		Metadata:   project.Metadata{Name: input.Name},
		Spec:       project.Spec{Backend: &project.TechnologySelection{Technology: input.Backend}},
	}
	if input.Frontend != "" && input.Frontend != "none" {
		if _, err := s.catalog.ValidateSelection(catalog.Frontend, input.Frontend); err != nil {
			return project.Manifest{}, "", err
		}
		manifest.Spec.Frontend = &project.TechnologySelection{Technology: input.Frontend}
	}
	if input.Database != "" && input.Database != "none" {
		if _, err := s.catalog.ValidateSelection(catalog.Database, input.Database); err != nil {
			return project.Manifest{}, "", err
		}
		manifest.Spec.Database = &project.TechnologySelection{Technology: input.Database}
	}
	if err := s.catalog.ValidateCombination(selectionIDs(manifest)...); err != nil {
		return project.Manifest{}, "", err
	}
	return manifest, path, nil
}

func (s ProjectService) CreateProject(input CreateProjectInput) (CreateProjectResult, error) {
	manifest, _, err := s.ValidateInput(input)
	if err != nil {
		return CreateProjectResult{}, err
	}
	path, err := project.Save(input.Destination, manifest)
	if err != nil {
		return CreateProjectResult{}, err
	}
	return CreateProjectResult{Manifest: manifest, ManifestPath: path}, nil
}

// CreateProjectFromManifest copies a validated configuration into a new project
// directory. It preserves the source's name and never overwrites a manifest.
func (s ProjectService) CreateProjectFromManifest(source, destination string) (CreateProjectResult, error) {
	manifest, err := s.LoadProject(source)
	if err != nil {
		return CreateProjectResult{}, err
	}
	path, err := project.Save(destination, manifest)
	if err != nil {
		return CreateProjectResult{}, err
	}
	return CreateProjectResult{Manifest: manifest, ManifestPath: path}, nil
}

// ValidateComponentsInput constructs a new-format manifest without writing it.
func (s ProjectService) ValidateComponentsInput(name, destination string, components []project.Component) (project.Manifest, error) {
	if _, err := s.ValidateProjectInfo(name, destination); err != nil {
		return project.Manifest{}, err
	}
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind,
		Metadata: project.Metadata{Name: name}, Spec: project.Spec{Components: append([]project.Component{}, components...)}}
	if err := manifest.Validate(); err != nil {
		return project.Manifest{}, err
	}
	for _, component := range manifest.CanonicalComponents() {
		if _, err := s.catalog.ValidateSelection(catalog.Category(component.Type), component.Technology); err != nil {
			return project.Manifest{}, fmt.Errorf("component %q: %w", component.Name, err)
		}
	}
	if err := s.catalog.ValidateCombination(selectionIDs(manifest)...); err != nil {
		return project.Manifest{}, err
	}
	return manifest, nil
}

func (s ProjectService) CreateComponentProject(name, destination string, components []project.Component) (CreateProjectResult, error) {
	manifest, err := s.ValidateComponentsInput(name, destination, components)
	if err != nil {
		return CreateProjectResult{}, err
	}
	path, err := project.Save(destination, manifest)
	if err != nil {
		return CreateProjectResult{}, err
	}
	return CreateProjectResult{Manifest: manifest, ManifestPath: path}, nil
}

// LoadProject validates both manifest structure and currently known technology IDs.
func (s ProjectService) LoadProject(path string) (project.Manifest, error) {
	manifest, err := project.Load(path)
	if err != nil {
		return project.Manifest{}, err
	}
	for _, component := range manifest.CanonicalComponents() {
		if _, err := s.catalog.ValidateSelection(catalog.Category(component.Type), component.Technology); err != nil {
			return project.Manifest{}, err
		}
	}
	if err := s.catalog.ValidateCombination(selectionIDs(manifest)...); err != nil {
		return project.Manifest{}, err
	}
	return manifest, nil
}

func selectionIDs(manifest project.Manifest) []string {
	components := manifest.CanonicalComponents()
	ids := make([]string, 0, len(components))
	for _, component := range components {
		ids = append(ids, component.Technology)
	}
	return ids
}
