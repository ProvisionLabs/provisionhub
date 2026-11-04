package tui

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/plugins"
	"github.com/ProvisionLabs/provisionhub/internal/registry"
	"github.com/charmbracelet/x/ansi"
)

type pluginService interface {
	List(context.Context) ([]plugins.Installation, error)
	Discover(context.Context) (registry.Snapshot, error)
	InstallFrom(context.Context, registry.Index, string, string) (plugins.Installation, error)
	Update(context.Context, string) (plugins.Installation, error)
	Remove(context.Context, string, string) error
}
type pluginResultMsg struct {
	request        uint64
	installed      []plugins.Installation
	snapshot       registry.Snapshot
	message        string
	err            error
	removedID      string
	removedVersion string
}
type pluginBrowser struct {
	service      pluginService
	installed    []plugins.Installation
	snapshot     registry.Snapshot
	mode         int // Available, Installed, Updates
	selected     int
	version      string
	query        string
	searching    bool
	details      bool
	confirmation bool
	busy         bool
	request      uint64
	cancel       context.CancelFunc
	message      string
	err          string
	selections   map[string]string
}

func newPluginBrowser(service pluginService) pluginBrowser {
	return pluginBrowser{service: service, mode: 1, selections: map[string]string{}}
}
func (p *pluginBrowser) stop() {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
}
func (p *pluginBrowser) start(action, id, version string) tea.Cmd {
	p.stop()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	p.cancel = cancel
	p.request++
	request := p.request
	p.busy = true
	p.err = ""
	p.confirmation = false
	p.message = map[string]string{"load": "Reading installed packages…", "refresh": "Fetching registry…", "install": "Downloading, verifying and installing…", "update": "Checking and installing update…", "remove": "Removing package…"}[action]
	service := p.service
	snapshot := p.snapshot
	return func() tea.Msg {
		defer cancel()
		result := pluginResultMsg{request: request, snapshot: snapshot}
		switch action {
		case "refresh":
			result.snapshot, result.err = service.Discover(ctx)
		case "install":
			var item plugins.Installation
			item, result.err = service.InstallFrom(ctx, snapshot.Index, id, version)
			if result.err == nil {
				result.message = "Installed " + item.PluginID + "@" + item.Version
			}
		case "update":
			var item plugins.Installation
			item, result.err = service.Update(ctx, id)
			if result.err == nil {
				result.message = "Installed " + item.PluginID + "@" + item.Version + "; older versions retained"
			}
		case "remove":
			result.err = service.Remove(ctx, id, version)
			if result.err == nil {
				result.message = "Removed " + id + "@" + version
				result.removedID = id
				result.removedVersion = version
			}
		}
		if result.err == nil {
			result.installed, result.err = service.List(ctx)
		}
		return result
	}
}
func (p *pluginBrowser) available() []registry.Plugin {
	result := p.snapshot.Index.Search(p.query)
	if p.mode == 2 {
		updates := application.PluginUpdates(p.snapshot.Index, p.installed)
		filtered := []registry.Plugin{}
		for _, candidate := range result {
			for _, update := range updates {
				if candidate.ID == update.PluginID {
					filtered = append(filtered, candidate)
				}
			}
		}
		return filtered
	}
	return result
}
func (p *pluginBrowser) count() int {
	if p.mode == 1 {
		return len(p.installed)
	}
	return len(p.available())
}
func (p *pluginBrowser) selection() (string, string) {
	if p.selected < 0 || p.selected >= p.count() {
		return "", ""
	}
	if p.mode == 1 {
		item := p.installed[p.selected]
		return item.PluginID, item.Version
	}
	candidate := p.available()[p.selected]
	version := p.version
	if version == "" {
		_, release, err := p.snapshot.Index.Resolve(candidate.ID, "", registry.CoreVersion)
		if err == nil {
			version = release.Version
		}
	}
	return candidate.ID, version
}
func (p *pluginBrowser) update(msg tea.Msg) (tea.Cmd, bool) {
	if result, ok := msg.(pluginResultMsg); ok {
		if result.request != p.request {
			return nil, false
		}
		p.busy = false
		p.cancel = nil
		if result.err != nil {
			p.err = result.err.Error()
			p.message = "Operation failed; inspect Installed for interrupted operations"
		} else {
			if p.selections[result.removedID] == result.removedVersion {
				delete(p.selections, result.removedID)
			}
			p.installed = result.installed
			p.snapshot = result.snapshot
			p.message = result.message
			p.selected = min(p.selected, max(0, p.count()-1))
		}
		return nil, false
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false
	}
	pressed := key.String()
	if p.busy {
		if pressed == "esc" {
			p.stop()
			p.message = "Cancelling…"
		}
		return nil, false
	}
	if p.searching {
		switch pressed {
		case "enter", "esc":
			p.searching = false
		case "backspace":
			r := []rune(p.query)
			if len(r) > 0 {
				p.query = string(r[:len(r)-1])
			}
		default:
			if len(key.Text) > 0 && len(p.query) < 128 {
				p.query += key.Text
			}
		}
		p.selected = 0
		return nil, false
	}
	if p.confirmation {
		p.confirmation = false
		if pressed == "y" {
			id, version := p.selection()
			return p.start("remove", id, version), false
		}
		return nil, false
	}
	switch pressed {
	case "t":
		p.mode = 2
		p.selected = 0
		p.version = ""
		p.details = false
		return p.start("refresh", "", ""), false
	case "esc", "q":
		p.stop()
		return nil, true
	case "a":
		p.mode = 0
		p.selected = 0
		p.version = ""
		p.details = false
		if p.snapshot.Index.SchemaVersion == 0 {
			return p.start("refresh", "", ""), false
		}
	case "l":
		p.mode = 1
		p.selected = 0
		p.details = false
		return p.start("load", "", ""), false
	case "u":
		if p.mode == 1 {
			id, _ := p.selection()
			if id != "" {
				return p.start("update", id, ""), false
			}
		} else {
			p.mode = 2
			p.selected = 0
			p.version = ""
			return p.start("refresh", "", ""), false
		}
	case "r":
		if p.mode == 1 {
			return p.start("load", "", ""), false
		}
		return p.start("refresh", "", ""), false
	case "/":
		if p.mode != 1 {
			p.searching = true
		}
	case "up", "k":
		p.selected = max(0, p.selected-1)
		p.version = ""
	case "down", "j":
		p.selected = min(max(0, p.count()-1), p.selected+1)
		p.version = ""
	case "enter":
		p.details = !p.details
	case "i":
		if p.mode != 1 {
			id, version := p.selection()
			if id != "" && version != "" {
				return p.start("install", id, version), false
			}
		}
	case "d":
		if p.mode == 1 && p.count() > 0 {
			p.confirmation = true
		}
	case "v":
		if p.mode != 1 && p.count() > 0 {
			candidate := p.available()[p.selected]
			versions := []string{}
			for _, release := range candidate.Versions {
				if registry.Compare(registry.CoreVersion, release.MinimumCoreVersion) >= 0 {
					versions = append(versions, release.Version)
				}
			}
			slices.SortFunc(versions, registry.Compare)
			_, current := p.selection()
			if len(versions) > 0 {
				p.version = versions[(slices.Index(versions, current)+1)%len(versions)]
			}
		}
	case "s":
		if p.mode == 1 && p.count() > 0 {
			item := p.installed[p.selected]
			if item.Status != "healthy" {
				p.err = "Only healthy installations can be selected"
			} else {
				p.selections[item.PluginID] = item.Version
				p.message = "Selected " + item.PluginID + "@" + item.Version + " for this session"
			}
		}
	}
	return nil, false
}
func (p pluginBrowser) view(width, height int) string {
	lines := []string{"Plugins — " + []string{"Available", "Installed", "Updates"}[p.mode], "a Available · l Installed · t Updates · r Refresh", "↑↓ navigate · enter Details · esc Back"}
	if p.mode != 1 {
		lines = append(lines, "/ Search · v Version · i Install", "Search: "+p.query)
		if p.snapshot.Stale {
			lines = append(lines, "STALE: "+p.snapshot.Warning)
		} else if p.snapshot.Warning != "" {
			lines = append(lines, p.snapshot.Warning)
		}
	} else {
		lines = append(lines, "u Update · d Remove · s Select version for generation")
	}
	if p.busy {
		lines = append(lines, p.message)
	} else {
		budget := max(1, height-len(lines)-7)
		start := max(0, p.selected-budget/2)
		end := min(p.count(), start+budget)
		for i := start; i < end; i++ {
			prefix := "  "
			if i == p.selected {
				prefix = "> "
			}
			if p.mode == 1 {
				item := p.installed[i]
				name := item.Name
				if name == "" {
					name = item.PluginID
				}
				update := ""
				for _, candidate := range application.PluginUpdates(p.snapshot.Index, p.installed) {
					if candidate.PluginID == item.PluginID {
						update = " → " + candidate.AvailableVersion
					}
				}
				lines = append(lines, fmt.Sprintf("%s%s %s%s [%s] %s", prefix, name, item.Version, update, item.Source, item.Status))
			} else {
				candidate := p.available()[i]
				_, release, err := p.snapshot.Index.Resolve(candidate.ID, "", registry.CoreVersion)
				version := "incompatible"
				if err == nil {
					version = release.Version
				}
				if i == p.selected && p.version != "" {
					version = p.version
				}
				status := "not installed"
				for _, item := range p.installed {
					if item.PluginID == candidate.ID {
						status = "installed " + item.Version
					}
				}
				lines = append(lines, fmt.Sprintf("%s%s (%s) %s — %s", prefix, candidate.Name, candidate.Category, version, status))
				if i == p.selected {
					lines = append(lines, "  "+candidate.Description)
				}
			}
		}
		if p.count() == 0 {
			lines = append(lines, "No plugins in this view. Available uses the official registry; Installed is offline.")
		}
		if p.details && p.count() > 0 {
			id, version := p.selection()
			lines = append(lines, id+"@"+version)
			if p.mode == 1 {
				lines = append(lines, p.installed[p.selected].Path)
			} else {
				candidate := p.available()[p.selected]
				for _, release := range candidate.Versions {
					lines = append(lines, release.Version+" · minimum core "+release.MinimumCoreVersion)
				}
			}
		}
		if p.confirmation {
			lines = append(lines, "Existing projects may need this version. Remove? y / any key cancels")
		}
		if p.message != "" {
			lines = append(lines, p.message)
		}
	}
	if p.err != "" {
		lines = append(lines, "Error: "+p.err)
	}
	return fitPluginLines(lines, width, height)
}
func fitPluginLines(lines []string, width, height int) string {
	// Keep errors/actions at the bottom visible, even in short terminals.
	if len(lines) > max(1, height) {
		lines = append(lines[:max(0, height-1)], lines[len(lines)-1])
	}
	for i, line := range lines {
		line = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return ' '
			}
			return r
		}, line)
		lines[i] = ansi.Truncate(line, max(1, width), "")
	}
	return strings.Join(lines, "\n")
}
