package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/planning"
)

func TestProjectPlanJSONIsStructuredAndDeterministic(t *testing.T) {
	path := planFixture(t)
	first, err := executeCLI("project", "plan", "--file", path, "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	var plan planning.Plan
	if err := json.Unmarshal([]byte(first), &plan); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, first)
	}
	if plan.ProjectName != "demo" || plan.Status != planning.StatusPlanned || plan.Executable || len(plan.Operations) != 3 || len(plan.Diagnostics) != 3 {
		t.Fatalf("JSON plan = %+v", plan)
	}
	for _, operation := range plan.Operations {
		if operation.Status != planning.OperationUnavailable || operation.TargetPaths == nil || operation.Dependencies == nil {
			t.Fatalf("JSON operation = %+v", operation)
		}
	}
	equivalent := filepath.Join(t.TempDir(), "equivalent.yaml")
	data := "# Same intent, different YAML ordering\nkind: Project\nspec:\n  database: {technology: postgresql}\n  frontend: {technology: nextjs}\n  backend: {technology: nestjs}\nmetadata: {name: demo}\napiVersion: provisionhub.dev/v1alpha1\n"
	if err := os.WriteFile(equivalent, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	text, err := executeCLI("project", "plan", "--file", path)
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		again, err := executeCLI("project", "plan", "--file", equivalent, "--output", "json")
		if err != nil || first != again {
			t.Fatalf("non-deterministic JSON: %v\n%s", err, again)
		}
		again, err = executeCLI("project", "plan", "--file", equivalent, "--output", "text")
		if err != nil || text != again {
			t.Fatalf("non-deterministic text: %v\n%s", err, again)
		}
	}
}

func TestProjectPlanErrorsReturnStructuredJSON(t *testing.T) {
	for _, test := range []struct {
		name, data string
		code       planning.DiagnosticCode
	}{
		{"missing", "", planning.ManifestLoadFailed},
		{"malformed", "metadata: [", planning.InvalidManifest},
		{"unknown technology", "apiVersion: provisionhub.dev/v1alpha1\nkind: Project\nmetadata: {name: demo}\nspec: {backend: {technology: missing}}\n", planning.InvalidTechnology},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "provisionhub.yaml")
			if test.name != "missing" {
				if err := os.WriteFile(path, []byte(test.data), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			out, err := executeCLI("project", "plan", "--file", path, "--output", "json")
			if err == nil {
				t.Fatal("failed planning must return an error (process exit 1)")
			}
			var plan planning.Plan
			if json.Unmarshal([]byte(out), &plan) != nil || plan.Status != planning.StatusInvalid || len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Code != test.code {
				t.Fatalf("failure JSON: %s; %v", out, err)
			}
		})
	}
}

func TestProjectPlanRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"project", "plan"},
		{"project", "plan", "--file", "missing", "--output", "yaml"},
		{"project", "plan", "--file", "missing", "extra"},
	} {
		if _, err := executeCLI(args...); err == nil {
			t.Fatalf("accepted invalid arguments: %v", args)
		}
	}
	out, err := executeCLI("project", "plan", "--help")
	if err != nil || !strings.Contains(out, "Exit 0") || !strings.Contains(out, "Exit 1") {
		t.Fatalf("exit semantics absent from help: %s; %v", out, err)
	}
}

type failedPlanWriter struct{}

func (failedPlanWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestProjectPlanPropagatesOutputErrors(t *testing.T) {
	path := planFixture(t)
	for _, format := range []string{"text", "json"} {
		cmd := NewRootCommand()
		cmd.SetOut(failedPlanWriter{})
		cmd.SetArgs([]string{"project", "plan", "--file", path, "--output", format})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "output unavailable") {
			t.Fatalf("%s output error = %v", format, err)
		}
	}
}
