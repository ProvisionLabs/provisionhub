package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
)

type browserService struct {
	items                           []plugins.Installation
	discoveries, installs, removals int
}

func (s *browserService) List(ctx context.Context) ([]plugins.Installation, error) {
	return s.items, ctx.Err()
}
func (s *browserService) Discover(ctx context.Context) (registry.Snapshot, error) {
	s.discoveries++
	return registry.Snapshot{Index: registry.Index{SchemaVersion: 1, Plugins: []registry.Plugin{{ID: "official/nestjs", ManifestID: "official-nestjs", Name: "NestJS", Description: "Backend generator", TechnologyID: "nestjs", Category: "backend", Versions: []registry.Release{{Version: "1.1.0", MinimumCoreVersion: "0.1.0"}, {Version: "1.2.0", MinimumCoreVersion: "0.1.0"}}}}}}, ctx.Err()
}
func (s *browserService) InstallFrom(ctx context.Context, _ registry.Index, id, version string) (plugins.Installation, error) {
	s.installs++
	item := plugins.Installation{PluginID: id, Version: version, Name: "NestJS", Source: "official", State: "ready", Status: "healthy"}
	if err := ctx.Err(); err != nil {
		return item, err
	}
	s.items = append(s.items, item)
	return item, nil
}
func (s *browserService) Update(ctx context.Context, id string) (plugins.Installation, error) {
	return s.InstallFrom(ctx, registry.Index{}, id, "1.2.0")
}
func (s *browserService) Remove(ctx context.Context, id, version string) error {
	s.removals++
	if err := ctx.Err(); err != nil {
		return err
	}
	s.items = nil
	return nil
}
func browserKey(p *pluginBrowser, text string) (tea.Cmd, bool) {
	return p.update(tea.KeyPressMsg{Code: []rune(text)[0], Text: text})
}
func finishBrowser(t *testing.T, p *pluginBrowser, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("missing async command")
	}
	p.update(cmd())
	if p.err != "" {
		t.Fatal(p.err)
	}
}
func TestBrowserDiscoveryInstallSelectionAndRemoval(t *testing.T) {
	service := &browserService{}
	p := newPluginBrowser(service)
	finishBrowser(t, &p, p.start("load", "", ""))
	if service.discoveries != 0 {
		t.Fatal("startup network")
	}
	cmd, _ := browserKey(&p, "a")
	if !p.busy || service.discoveries != 0 {
		t.Fatal("blocking discovery")
	}
	finishBrowser(t, &p, cmd)
	browserKey(&p, "v")
	_, version := p.selection()
	if version != "1.1.0" {
		t.Fatal("version choice", version)
	}
	browserKey(&p, "/")
	browserKey(&p, "n")
	p.update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if p.query != "n" || p.searching {
		t.Fatal("search")
	}
	cmd, _ = browserKey(&p, "i")
	finishBrowser(t, &p, cmd)
	if service.installs != 1 {
		t.Fatal("install")
	}
	cmd, _ = browserKey(&p, "l")
	finishBrowser(t, &p, cmd)
	browserKey(&p, "s")
	if p.selections["official/nestjs"] != "1.1.0" {
		t.Fatal("session selection")
	}
	browserKey(&p, "d")
	if !p.confirmation || service.removals != 0 {
		t.Fatal("remove without confirmation")
	}
	browserKey(&p, "n")
	if service.removals != 0 {
		t.Fatal("cancel removal")
	}
	browserKey(&p, "d")
	cmd, _ = browserKey(&p, "y")
	finishBrowser(t, &p, cmd)
	if service.removals != 1 || len(p.selections) != 0 {
		t.Fatal("remove")
	}
}
func TestBrowserCancellationStaleResultAndSizing(t *testing.T) {
	p := newPluginBrowser(&browserService{})
	cmd := p.start("refresh", "", "")
	p.update(tea.KeyPressMsg{Code: tea.KeyEscape})
	p.update(cmd())
	if !strings.Contains(p.err, "canceled") {
		t.Fatal(p.err)
	}
	request := p.request
	p.update(pluginResultMsg{request: request - 1, err: errors.New("obsolete")})
	if strings.Contains(p.err, "obsolete") {
		t.Fatal("stale result")
	}
	p.installed = []plugins.Installation{{Name: strings.Repeat("界", 100), Status: "healthy"}}
	p.mode = 1
	for _, size := range [][2]int{{80, 24}, {20, 5}, {1, 1}} {
		view := p.view(size[0], size[1])
		if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] {
			t.Fatal("overflow", size, view)
		}
	}
}
