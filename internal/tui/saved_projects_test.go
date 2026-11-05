package tui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/localdb"
	"github.com/ProvisionLabs/provisionhub/internal/project"
	"github.com/ProvisionLabs/provisionhub/internal/projectlock"
)

func savedKey(b *savedBrowser, key string) tea.Cmd {
	code := rune(0)
	if len(key) == 1 {
		code = []rune(key)[0]
	} else if key == "enter" {
		code = tea.KeyEnter
	} else if key == "esc" {
		code = tea.KeyEscape
	}
	cmd, _ := b.update(tea.KeyPressMsg{Code: code, Text: func() string {
		if len(key) == 1 {
			return key
		}
		return ""
	}()})
	return cmd
}

func deliverSaved(b *savedBrowser, cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}
	result, ok := cmd().(savedResultMsg)
	if !ok {
		return nil
	}
	return b.receive(result)
}

func TestMyProjectsRegistrationDetailsRelocationAndForget(t *testing.T) {
	root := t.TempDir()
	service := application.NewProjectService()
	b := newSavedBrowser(application.SavedProjects{DatabasePath: filepath.Join(root, "data.db"), Projects: service})
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"}, Spec: project.Spec{Components: []project.Component{{Name: "api", Type: "backend", Technology: "nestjs"}}}}
	file, err := project.Save(filepath.Join(root, "project"), manifest)
	if err != nil {
		t.Fatal(err)
	}
	deliverSaved(&b, b.load())
	if !strings.Contains(b.view(80, 24), "No saved projects") {
		t.Fatal("empty list missing")
	}
	savedKey(&b, "r")
	if b.state != "register" {
		t.Fatal("register action missing")
	}
	b.input.SetValue(file)
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "enter")))
	if len(b.views) != 1 || b.views[0].Manifest.Metadata.Name != "demo" {
		t.Fatalf("registration/list: %+v %s", b.views, b.err)
	}
	deliverSaved(&b, savedKey(&b, "enter"))
	if b.state != "details" || !strings.Contains(b.view(80, 24), "api") {
		t.Fatalf("details: %s", b.view(80, 24))
	}
	if b.current.Reference.LastOpenedAt == nil {
		t.Fatal("last opened time missing")
	}
	deliverSaved(&b, savedKey(&b, "esc"))
	if err := os.Rename(filepath.Dir(file), filepath.Join(root, "moved")); err != nil {
		t.Fatal(err)
	}
	deliverSaved(&b, b.load())
	if b.views[0].ManifestError == nil {
		t.Fatal("moved project not shown unavailable")
	}
	savedKey(&b, "m")
	b.input.SetValue(filepath.Join(root, "moved", project.ManifestName))
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "enter")))
	if b.views[0].ManifestError != nil {
		t.Fatal("relocation failed")
	}
	savedKey(&b, "f")
	if _, err := os.Stat(filepath.Join(root, "moved", project.ManifestName)); err != nil {
		t.Fatal(err)
	}
	savedKey(&b, "n")
	if len(b.views) != 1 {
		t.Fatal("cancel forgot project")
	}
	savedKey(&b, "f")
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "y")))
	if len(b.views) != 0 {
		t.Fatalf("forget did not clear list: %+v", b.views)
	}
	if _, err := os.Stat(filepath.Join(root, "moved", project.ManifestName)); err != nil {
		t.Fatalf("forget deleted files: %v", err)
	}
	views, err := b.saved.List(context.Background())
	if err != nil || len(views) != 0 {
		t.Fatalf("database still registered: %v %+v", err, views)
	}
}

func TestMyProjectsLockPreviewRequiresConfirmation(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "plugins", "nestjs")
	if err := os.MkdirAll(filepath.Join(pluginDir, "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	definition := `apiVersion: provisionhub.dev/plugin/v1alpha1
id: official-nestjs
version: 1.0.0
technologyId: nestjs
category: backend
name: NestJS
description: Local test generator
capability: generate_backend
files:
  - type: template
    source: templates/package.tmpl
    destination: package.json
`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(definition), 0644); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(pluginDir, "templates", "package.tmpl")
	if err := os.WriteFile(template, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	service, err := application.NewProjectServiceWithPlugins(filepath.Join(root, "plugins"))
	if err != nil {
		t.Fatal(err)
	}
	b := newSavedBrowser(application.SavedProjects{DatabasePath: filepath.Join(root, "data.db"), Projects: service})
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "demo"}, Spec: project.Spec{Backend: &project.TechnologySelection{Technology: "nestjs"}}}
	file, err := project.Save(filepath.Join(root, "demo"), manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.saved.Register(context.Background(), file); err != nil {
		t.Fatal(err)
	}
	deliverSaved(&b, b.load())
	deliverSaved(&b, savedKey(&b, "enter"))
	savedKey(&b, "l")
	deliverSaved(&b, savedKey(&b, "c"))
	if b.state != "lockPreview" {
		t.Fatalf("preview: %s %s", b.state, b.err)
	}
	lockPath, err := projectlock.Path(file)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("preview wrote lock")
	}
	savedKey(&b, "enter")
	savedKey(&b, "n")
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("cancel wrote lock")
	}
	savedKey(&b, "enter")
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "y")))
	initial, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	generated := filepath.Join(root, "demo", "package.json")
	deliverSaved(&b, savedKey(&b, "p"))
	if b.state != "plan" || !b.plan.Executable {
		t.Fatalf("saved plan: %s %+v %s", b.state, b.plan, b.err)
	}
	if _, err := os.Stat(generated); !os.IsNotExist(err) {
		t.Fatal("planning wrote output")
	}
	savedKey(&b, "g")
	if b.state != "applyConfirm" {
		t.Fatalf("apply confirmation missing: %s %s", b.state, b.err)
	}
	savedKey(&b, "n")
	if _, err := os.Stat(generated); !os.IsNotExist(err) {
		t.Fatal("cancelled apply wrote output")
	}
	savedKey(&b, "g")
	deliverSaved(&b, savedKey(&b, "y"))
	if _, err := os.Stat(generated); err != nil {
		t.Fatalf("confirmed apply failed: %v %s %+v", err, b.err, b.plan.Operations)
	}
	savedKey(&b, "l")
	if err := os.WriteFile(template, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	deliverSaved(&b, savedKey(&b, "u"))
	if b.state != "lockPreview" {
		t.Fatalf("update preview: %s %s", b.state, b.err)
	}
	savedKey(&b, "enter")
	savedKey(&b, "n")
	still, err := os.ReadFile(lockPath)
	if err != nil || !bytes.Equal(initial, still) {
		t.Fatal("cancelled update changed lock")
	}
	savedKey(&b, "enter")
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "y")))
	updated, err := os.ReadFile(lockPath)
	if err != nil || bytes.Equal(initial, updated) {
		t.Fatalf("confirmed update did not change lock: %v", err)
	}
}

func TestSavedApplyCannotQuitDuringPublication(t *testing.T) {
	m := newTestModel(t)
	m.nav.current = openProject
	m.savedBrowser.state = "applyConfirm"
	m.savedBrowser.busy = true
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd != nil || m.nav.current != openProject {
		t.Fatal("quit interrupted saved project apply")
	}
}

func TestUnavailableCleanupReviewCancelSelectionAndRendering(t *testing.T) {
	root := t.TempDir()
	service := application.NewProjectService()
	b := newSavedBrowser(application.SavedProjects{DatabasePath: filepath.Join(root, "data.db"), Projects: service})
	manifest := project.Manifest{APIVersion: project.APIVersion, Kind: project.Kind, Metadata: project.Metadata{Name: "available"}, Spec: project.Spec{Backend: &project.TechnologySelection{Technology: "nestjs"}}}
	files := []string{}
	ids := []string{}
	for _, name := range []string{"available", "first temporary project with spaces", "second temporary project with spaces"} {
		file, err := project.Save(filepath.Join(root, name), manifest)
		if err != nil {
			t.Fatal(err)
		}
		ref, err := b.saved.Register(context.Background(), file)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
		ids = append(ids, ref.ID)
	}
	firstMoved := filepath.Join(root, "first moved")
	secondMoved := filepath.Join(root, "second moved")
	if err := os.Rename(filepath.Dir(files[1]), firstMoved); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Dir(files[2]), secondMoved); err != nil {
		t.Fatal(err)
	}
	deliverSaved(&b, b.load())
	for _, size := range [][2]int{{80, 20}, {54, 10}, {24, 6}, {5, 3}, {1, 1}} {
		view := b.view(size[0], size[1])
		for _, line := range strings.Split(view, "\n") {
			if lipgloss.Width(line) > size[0] {
				t.Fatalf("list overflow %v: %q", size, line)
			}
		}
		if strings.Contains(view, "read manifest") {
			t.Fatal("long error leaked into list")
		}
	}
	if !strings.Contains(b.view(80, 20), "Unavailable") || !strings.Contains(b.view(80, 20), "Available") {
		t.Fatal("mixed states missing")
	}
	deliverSaved(&b, savedKey(&b, "c"))
	if b.state != "cleanup" || len(b.candidates) != 2 || len(b.selectedCleanupIDs()) != 0 {
		t.Fatalf("preview: %s %+v", b.state, b.candidates)
	}
	savedKey(&b, "enter")
	if b.state != "cleanup" {
		t.Fatal("empty selection confirmed")
	}
	savedKey(&b, "a")
	if len(b.selectedCleanupIDs()) != 2 {
		t.Fatal("select all missed a candidate")
	}
	savedKey(&b, "x")
	if len(b.selectedCleanupIDs()) != 0 {
		t.Fatal("clear selection failed")
	}
	savedKey(&b, " ")
	if len(b.selectedCleanupIDs()) != 1 {
		t.Fatal("space did not select candidate")
	}
	savedKey(&b, "enter")
	if b.state != "cleanupConfirm" {
		t.Fatal("confirmation missing")
	}
	savedKey(&b, "n")
	if len(b.views) != 3 {
		t.Fatal("cancelled cleanup removed records")
	}
	savedKey(&b, "enter")
	deliverSaved(&b, deliverSaved(&b, savedKey(&b, "y")))
	views, err := b.saved.List(context.Background())
	if err != nil || len(views) != 2 {
		t.Fatalf("selective cleanup: %+v %v", views, err)
	}
	if _, err := os.Stat(filepath.Join(firstMoved, project.ManifestName)); err != nil {
		t.Fatalf("cleanup deleted files: %v", err)
	}
	if _, err := b.saved.Open(context.Background(), ids[0]); err != nil {
		t.Fatalf("available project removed: %v", err)
	}
	if err := b.saved.Relocate(context.Background(), ids[2], secondMoved); err != nil {
		t.Fatal(err)
	}
	deliverSaved(&b, b.load())
	if len(b.views) != 2 || b.views[1].ManifestError != nil {
		t.Fatalf("remaining project not recoverable: %+v", b.views)
	}
	deliverSaved(&b, savedKey(&b, "c"))
	if len(b.candidates) != 0 || !strings.Contains(b.view(80, 20), "No unavailable projects") {
		t.Fatal("available projects offered for cleanup")
	}
	savedKey(&b, "esc")
	if b.state != "list" {
		t.Fatal("cleanup back navigation failed")
	}
	longWindows := application.SavedProjectView{Reference: localdb.SavedProject{ManifestPath: `C:\Users\Example\AppData\Local\Temp\1\Test Project With Spaces\001\new project\provisionhub.yaml`}, ManifestError: errors.New(strings.Repeat("permission denied: ", 30))}
	b.views = append(b.views, longWindows)
	if strings.Contains(b.view(80, 20), "permission denied") {
		t.Fatal("raw Windows error rendered in list")
	}
	for _, line := range strings.Split(b.view(24, 7), "\n") {
		if lipgloss.Width(line) > 24 {
			t.Fatalf("narrow Windows row overflow: %q", line)
		}
	}
	b.current = longWindows
	b.state = "details"
	if !strings.Contains(b.view(80, 20), "Error:") || !strings.Contains(b.view(80, 20), "Manifest:") {
		t.Fatal("detailed unavailable diagnostics missing")
	}
	preview := b
	preview.state = "cleanupConfirm"
	preview.candidates = make([]application.SavedProjectView, 30)
	for i := range preview.candidates {
		preview.candidates[i] = longWindows
	}
	preview.cleanupSelected = map[string]bool{}
	if !strings.Contains(preview.view(60, 8), "y confirm") {
		t.Fatalf("cleanup confirmation scrolled out of view (error=%q): %q", preview.err, preview.view(60, 8))
	}
}
