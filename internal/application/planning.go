package application

import (
	"fmt"
	"reflect"

	"github.com/ProvisionLabs/provisionhub/internal/execution"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

// PlanSpecification plans directly from an in-memory project description.
func (s ProjectService) PlanSpecification(spec project.Specification) (planning.Plan, error) {
	loaded, err := s.loadPlugins()
	if err != nil {
		return planning.Plan{Status: planning.StatusInvalid, Technologies: []planning.Technology{}, Operations: []planning.Operation{}, Diagnostics: []planning.Diagnostic{{Severity: planning.SeverityError, Code: planning.InvalidPlugin, Message: err.Error()}}}, err
	}
	plan := planning.BuildWithPlugins(spec, s.catalog, loaded)
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Severity == planning.SeverityError {
			return plan, fmt.Errorf("cannot construct generation plan: %s", diagnostic.Message)
		}
	}
	return plan, nil
}

// PreflightSpecification checks the reviewed plan and output without writing.
func (s ProjectService) PreflightSpecification(spec project.Specification, destination string, reviewed planning.Plan) error {
	loaded, err := s.freshSpecification(spec, reviewed)
	if err != nil {
		return err
	}
	return execution.Preflight(reviewed, spec, loaded, destination)
}

// ApplySpecification reloads plugin content and repeats the reviewed plan check.
func (s ProjectService) ApplySpecification(spec project.Specification, destination string, reviewed planning.Plan) (execution.Result, error) {
	loaded, err := s.freshSpecification(spec, reviewed)
	if err != nil {
		return execution.Result{}, err
	}
	return execution.Apply(reviewed, spec, loaded, destination)
}

func (s ProjectService) freshSpecification(spec project.Specification, reviewed planning.Plan) (plugins.Set, error) {
	loaded, err := s.loadPlugins()
	if err != nil {
		return plugins.Set{}, err
	}
	fresh := planning.BuildWithPlugins(spec, s.catalog, loaded)
	if !fresh.Executable || !reflect.DeepEqual(reviewed, fresh) {
		return plugins.Set{}, fmt.Errorf("generation plan changed or is unavailable; review a new plan before applying")
	}
	return loaded, nil
}
