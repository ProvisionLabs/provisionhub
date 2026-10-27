package application

import (
	"errors"
	"fmt"
	"os"

	"github.com/ProvisionLabs/provisionhub/internal/planning"
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
	plan := planning.Build(manifest, s.catalog)
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Severity == planning.SeverityError {
			return plan, fmt.Errorf("cannot construct generation plan: %s", diagnostic.Message)
		}
	}
	return plan, nil
}
