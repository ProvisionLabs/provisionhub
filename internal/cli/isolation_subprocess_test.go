package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/project"
)

func TestProjectInitSubprocessUsesOnlyIsolatedDatabase(t *testing.T) {
	root := t.TempDir()
	sentinel := application.SavedProjects{DatabasePath: filepath.Join(root, "sentinel", "provisionhub.db"), Projects: application.NewProjectService()}
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "legitimate"}, Spec: project.Spec{Backend: &project.TechnologySelection{Technology: "nestjs"}}}
	legitimate, err := project.Save(filepath.Join(root, "legitimate"), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sentinel.Register(context.Background(), legitimate); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(sentinel.DatabasePath)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "provisionhub")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", binary, "./cmd/provisionhub")
	build.Dir = repo
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build subprocess CLI: %v\n%s", err, output)
	}
	childHome := filepath.Join(root, "child-home")
	if err := os.MkdirAll(childHome, 0700); err != nil {
		t.Fatal(err)
	}
	childData := filepath.Join(root, "child-data")
	childEnv := append(os.Environ(), "PROVISIONHUB_DATA_DIR="+childData, "XDG_DATA_HOME="+childHome, "XDG_CONFIG_HOME="+childHome, "APPDATA="+childHome, "LOCALAPPDATA="+childHome, "USERPROFILE="+childHome, "HOME="+childHome)
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(binary, args...)
		cmd.Env = childEnv
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("child command %v: %v\n%s", args, err, output)
		}
		return string(output)
	}
	created := filepath.Join(root, "temporary")
	run("project", "init", "--name", "temporary", "--path", created, "--backend", "nestjs")
	listed := run("project", "list") // a second OS process reopens the child's database
	if !strings.Contains(listed, "temporary") || strings.Contains(listed, "legitimate") {
		t.Fatalf("child list: %s", listed)
	}
	isolated := application.SavedProjects{DatabasePath: filepath.Join(childData, "provisionhub.db"), Projects: application.NewProjectService()}
	views, err := isolated.List(context.Background())
	if err != nil || len(views) != 1 || views[0].Manifest.Metadata.Name != "temporary" {
		t.Fatalf("isolated database: %+v %v", views, err)
	}
	after, err := os.ReadFile(sentinel.DatabasePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("sentinel database changed: %v", err)
	}
	sentinelViews, err := sentinel.List(context.Background())
	if err != nil || len(sentinelViews) != 1 || sentinelViews[0].Manifest.Metadata.Name != "legitimate" {
		t.Fatalf("sentinel contents changed: %+v %v", sentinelViews, err)
	}
}
