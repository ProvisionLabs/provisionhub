package application

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/planning"
)

func projectSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	snapshot := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Directory timestamps can change during read-only fixture inspection on Windows.
		// Tree shape, permissions, and file contents/modification times remain comparable.
		value := info.Mode().String()
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value += fmt.Sprintf(":%s:%s", info.ModTime(), data)
		}
		snapshot[relative] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestPlanProjectLoadsWithoutChangingFiles(t *testing.T) {
	root := t.TempDir()
	service := NewProjectService()
	created, err := service.CreateProject(validInput(filepath.Join(root, "my project café")))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	before := projectSnapshot(t, root)
	plan, err := service.PlanProject(created.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != planning.StatusPlanned || plan.Executable || len(plan.Operations) != 3 || len(plan.Diagnostics) != 3 {
		t.Fatalf("plan = %+v", plan)
	}
	for _, diagnostic := range plan.Diagnostics {
		if diagnostic.Code != planning.MissingOperationProvider || diagnostic.Severity != planning.SeverityWarning {
			t.Fatalf("diagnostic = %+v", diagnostic)
		}
	}
	fromMemory, err := service.PlanManifest(created.Manifest)
	if err != nil || !reflect.DeepEqual(plan, fromMemory) {
		t.Fatalf("shared plan = %+v, %v", fromMemory, err)
	}
	if after := projectSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatalf("planning changed files: before=%v, after=%v", before, after)
	}
}
