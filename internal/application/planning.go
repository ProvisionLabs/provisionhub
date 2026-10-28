package application

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/ProvisionLabs/provisionhub/internal/execution"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// PlanProject reads a manifest without inspecting or changing its destination.
func (s ProjectService) PlanProject(path string) (planning.Plan, error) {
	manifest, err := project.Load(path)
	if err != nil {
		code := planning.InvalidManifest
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			code = planning.ManifestLoadFailed
		}
		return planning.Plan{
			Status: planning.StatusInvalid, Technologies: []planning.Technology{}, Operations: []planning.Operation{},
			Diagnostics: []planning.Diagnostic{{Severity: planning.SeverityError, Code: code, Message: err.Error()}},
		}, err
	}
	return s.PlanManifest(manifest)
}

// PlanManifest shares pure planning between saved projects and unsaved TUI input.
func (s ProjectService) PlanManifest(manifest project.Manifest) (planning.Plan, error) {
	loaded, err := plugins.LoadDirectory(s.pluginDirectory, s.catalog)
	if err != nil {
		return planning.Plan{Status: planning.StatusInvalid, Technologies: []planning.Technology{}, Operations: []planning.Operation{}, Diagnostics: []planning.Diagnostic{{Severity: planning.SeverityError, Code: planning.InvalidPlugin, Message: err.Error()}}}, err
	}
	plan := planning.BuildWithPlugins(manifest, s.catalog, loaded)
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Severity == planning.SeverityError {
			return plan, fmt.Errorf("cannot construct generation plan: %s", diagnostic.Message)
		}
	}
	return plan, nil
}

// PreflightProject checks that a reviewed plan still matches current inputs
// and that no generated target already exists. It performs no writes.
func (s ProjectService) PreflightProject(file string, reviewed planning.Plan) error {
	manifest, loaded, err := s.freshInputs(file, reviewed)
	if err != nil {
		return err
	}
	return execution.Preflight(reviewed, manifest, loaded, filepath.Dir(file))
}

// ApplyProject reloads the manifest and plugin files before applying a reviewed plan.
func (s ProjectService) ApplyProject(file string, reviewed planning.Plan) (execution.Result, error) {
	manifest, loaded, err := s.freshInputs(file, reviewed)
	if err != nil {
		return execution.Result{}, err
	}
	return execution.Apply(reviewed, manifest, loaded, filepath.Dir(file))
}

func (s ProjectService) freshInputs(file string, reviewed planning.Plan) (project.Manifest, plugins.Set, error) {
	manifest, err := s.LoadProject(file)
	if err != nil {
		return project.Manifest{}, plugins.Set{}, err
	}
	loaded, err := plugins.LoadDirectory(s.pluginDirectory, s.catalog)
	if err != nil {
		return project.Manifest{}, plugins.Set{}, err
	}
	fresh := planning.BuildWithPlugins(manifest, s.catalog, loaded)
	if !fresh.Executable || !reflect.DeepEqual(reviewed, fresh) {
		return project.Manifest{}, plugins.Set{}, fmt.Errorf("generation plan changed or is unavailable; review a new plan before applying")
	}
	return manifest, loaded, nil
}
