package planning

import (
	"fmt"
	"slices"
	"strings"
)

// orderOperations uses dependency-first ordering, with lexical IDs as the tie-breaker.
// It copies collection fields so planning never mutates contributed definitions.
func orderOperations(input []Operation) ([]Operation, []Diagnostic) {
	operations := slices.Clone(input)
	slices.SortFunc(operations, func(a, b Operation) int { return strings.Compare(a.ID, b.ID) })
	known := make(map[string]bool, len(operations))
	diagnostics := []Diagnostic{}
	for i := range operations {
		operation := &operations[i]
		if strings.TrimSpace(operation.ID) == "" || known[operation.ID] {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError, Code: InvalidOperation, OperationID: operation.ID,
				Message: fmt.Sprintf("Operation ID %q must be nonempty and unique.", operation.ID),
			})
		}
		known[operation.ID] = true
		operation.Dependencies = append([]string{}, operation.Dependencies...)
		slices.Sort(operation.Dependencies)
		operation.Dependencies = slices.Compact(operation.Dependencies)
		operation.TargetPaths = append([]string{}, operation.TargetPaths...)
		slices.Sort(operation.TargetPaths)
		operation.TargetPaths = slices.Compact(operation.TargetPaths)
	}
	for _, operation := range operations {
		for _, dependency := range operation.Dependencies {
			if !known[dependency] {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: SeverityError, Code: InvalidDependency, OperationID: operation.ID,
					Message: fmt.Sprintf("Operation %q depends on unknown operation %q.", operation.ID, dependency),
				})
			}
		}
	}
	if len(diagnostics) != 0 {
		return []Operation{}, diagnostics
	}
	ordered := []Operation{}
	done := make(map[string]bool, len(operations))
	for len(ordered) < len(operations) {
		found := false
		for _, operation := range operations {
			if done[operation.ID] {
				continue
			}
			ready := true
			for _, dependency := range operation.Dependencies {
				if !done[dependency] {
					ready = false
					break
				}
			}
			if ready {
				ordered = append(ordered, operation)
				done[operation.ID] = true
				found = true
				break
			}
		}
		if !found {
			var remaining []string
			for _, operation := range operations {
				if !done[operation.ID] {
					remaining = append(remaining, operation.ID)
				}
			}
			return []Operation{}, []Diagnostic{{
				Severity: SeverityError, Code: DependencyCycle,
				Message: "A dependency cycle prevents ordering these remaining operations: " + strings.Join(remaining, ", ") + ".",
			}}
		}
	}
	return ordered, diagnostics
}

func finalizePlan(plan Plan) Plan {
	operations, diagnostics := orderOperations(plan.Operations)
	plan.Operations = operations
	plan.Diagnostics = append(plan.Diagnostics, diagnostics...)
	fileTargets := map[string]string{}
	for _, operation := range plan.Operations {
		if operation.Type != GenerateFile {
			continue
		}
		for _, target := range operation.TargetPaths {
			key := strings.ToLower(target)
			for existing, owner := range fileTargets {
				if existing == key || strings.HasPrefix(existing, key+"/") || strings.HasPrefix(key, existing+"/") {
					plan.Diagnostics = append(plan.Diagnostics, Diagnostic{
						Severity: SeverityError, Code: InvalidOperation, OperationID: operation.ID,
						Message: fmt.Sprintf("Output path %q conflicts with operation %q.", target, owner),
					})
				}
			}
			fileTargets[key] = operation.ID
		}
	}
	plan.Executable = plan.Status == StatusPlanned && len(plan.Operations) > 0
	for _, operation := range plan.Operations {
		if operation.Status != OperationSupported {
			plan.Executable = false
		}
	}
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Severity == SeverityError {
			plan.Status = StatusInvalid
			plan.Executable = false
		}
	}
	slices.SortFunc(plan.Diagnostics, func(a, b Diagnostic) int {
		key := func(d Diagnostic) string {
			return strings.Join([]string{string(d.Severity), string(d.Code), d.OperationID, d.ComponentID, d.Technology, d.Field, d.Message}, "\x00")
		}
		return strings.Compare(key(a), key(b))
	})
	return plan
}
