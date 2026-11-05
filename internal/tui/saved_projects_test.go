package tui

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
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
	b := newSavedBrowser(service)
	b.saved.DatabasePath = filepath.Join(root, "data.db")
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
	b := newSavedBrowser(service)
	b.saved.DatabasePath = filepath.Join(root, "data.db")
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
