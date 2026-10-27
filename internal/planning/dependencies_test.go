package planning

import (
	"reflect"
	"slices"
	"testing"
)

func TestOrderOperationsUsesDependenciesAndStableIDs(t *testing.T) {
	input := []Operation{
		{ID: "z", Dependencies: []string{"m", "a", "m"}, TargetPaths: []string{"z-target", "a-target", "z-target"}},
		{ID: "m", Dependencies: []string{"a"}},
		{ID: "b"},
		{ID: "a"},
	}
	before := append([]Operation(nil), input...)
	for i := range before {
		before[i].Dependencies = slices.Clone(input[i].Dependencies)
		before[i].TargetPaths = slices.Clone(input[i].TargetPaths)
	}
	ordered, diagnostics := orderOperations(input)
	var ids []string
	for _, operation := range ordered {
		ids = append(ids, operation.ID)
	}
	if !reflect.DeepEqual(ids, []string{"a", "b", "m", "z"}) || len(diagnostics) != 0 {
		t.Fatalf("order = %v; diagnostics = %+v", ids, diagnostics)
	}
	if !reflect.DeepEqual(ordered[3].Dependencies, []string{"a", "m"}) {
		t.Fatalf("dependencies not canonicalized: %v", ordered[3].Dependencies)
	}
	if !reflect.DeepEqual(ordered[3].TargetPaths, []string{"a-target", "z-target"}) {
		t.Fatalf("target paths not canonicalized: %v", ordered[3].TargetPaths)
	}
	if !reflect.DeepEqual(input, before) {
		t.Fatal("ordering mutated the input")
	}
	for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
		input[i], input[j] = input[j], input[i]
	}
	again, _ := orderOperations(input)
	if !reflect.DeepEqual(ordered, again) {
		t.Fatal("input order affected the result")
	}
	ordered[3].Dependencies[0] = "changed"
	ordered[3].TargetPaths[0] = "changed"
	if !reflect.DeepEqual(input[len(input)-1], before[0]) {
		t.Fatal("returned operations share caller-owned collection storage")
	}
}

func TestInvalidOperationGraphsProduceOrderedErrorDiagnostics(t *testing.T) {
	for _, test := range []struct {
		name       string
		operations []Operation
		code       DiagnosticCode
	}{
		{"missing dependency", []Operation{{ID: "a", Dependencies: []string{"missing"}}}, InvalidDependency},
		{"cycle", []Operation{{ID: "b", Dependencies: []string{"a"}}, {ID: "a", Dependencies: []string{"b"}}}, DependencyCycle},
		{"self cycle", []Operation{{ID: "a", Dependencies: []string{"a"}}}, DependencyCycle},
		{"duplicate ID", []Operation{{ID: "a"}, {ID: "a"}}, InvalidOperation},
		{"empty ID", []Operation{{ID: ""}}, InvalidOperation},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan := finalizePlan(Plan{Operations: test.operations, Status: StatusPlanned, Executable: true})
			if plan.Status != StatusInvalid || plan.Executable || len(plan.Operations) != 0 || len(plan.Diagnostics) == 0 {
				t.Fatalf("invalid graph = %+v", plan)
			}
			for _, diagnostic := range plan.Diagnostics {
				if diagnostic.Code != test.code || diagnostic.Severity != SeverityError {
					t.Fatalf("diagnostic = %+v", diagnostic)
				}
			}
		})
	}
}
