package tui

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ProvisionLabs/provisionhub/internal/application"
	"github.com/ProvisionLabs/provisionhub/internal/execution"
	"github.com/ProvisionLabs/provisionhub/internal/planning"
)

type savedResultMsg struct {
	action  string
	views   []application.SavedProjectView
	view    application.SavedProjectView
	plan    planning.Plan
	preview application.ProjectLockPreview
	result  execution.Result
	err     error
}

type savedBrowser struct {
	service     application.ProjectService
	saved       application.SavedProjects
	initErr     error
	state       string
	returnState string
	views       []application.SavedProjectView
	selected    int
	current     application.SavedProjectView
	targetID    string
	input       textinput.Model
	plan        planning.Plan
	preview     application.ProjectLockPreview
	offset      int
	busy        bool
	notice      string
	err         string
}

func newSavedBrowser(service application.ProjectService) savedBrowser {
	saved, err := application.NewSavedProjects(service)
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "./project/provisionhub.yaml"
	input.SetWidth(60)
	return savedBrowser{service: service, saved: saved, initErr: err, state: "list", input: input}
}

func (b *savedBrowser) load() tea.Cmd {
	b.busy = true
	saved, initErr := b.saved, b.initErr
	return func() tea.Msg {
		if initErr != nil {
			return savedResultMsg{action: "list", err: initErr}
		}
		views, err := saved.List(context.Background())
		return savedResultMsg{action: "list", views: views, err: err}
	}
}

func (b *savedBrowser) receive(message savedResultMsg) tea.Cmd {
	b.busy = false
	if message.action == "open" && message.err != nil {
		b.current = message.view
		b.offset = 0
		b.state = "details"
		b.err = message.err.Error()
		return nil
	}
	if message.action == "plan" && message.err != nil {
		b.plan = message.plan
		b.state = "plan"
		b.err = message.err.Error()
		return nil
	}
	if message.err != nil {
		b.err = message.err.Error()
		return nil
	}
	b.err = ""
	switch message.action {
	case "list":
		b.views = message.views
		b.selected = max(0, min(b.selected, len(b.views)-1))
		b.offset = 0
		b.state = "list"
	case "open", "refresh":
		b.current = message.view
		b.offset = 0
		b.state = "details"
	case "register", "relocate", "forget":
		b.notice = strings.Title(message.action) + " completed"
		return b.load()
	case "plan":
		b.plan = message.plan
		b.offset = 0
		b.state = "plan"
	case "preview":
		b.preview = message.preview
		b.offset = 0
		b.state = "lockPreview"
	case "lockWrite", "restore":
		b.notice = map[string]string{"lockWrite": "Plugin lock saved", "restore": "Locked plugins restored"}[message.action]
		return b.refresh()
	case "check":
		b.notice = "Plugin lock is valid"
	case "openDirectory":
		b.notice = "Opened project directory"
	case "apply":
		b.notice = fmt.Sprintf("Generated %d files", len(message.result.Created))
		b.state = "details"
	}
	return nil
}

func (b *savedBrowser) refresh() tea.Cmd {
	id, saved := b.current.Reference.ID, b.saved
	b.busy = true
	return func() tea.Msg {
		view, err := saved.Open(context.Background(), id)
		return savedResultMsg{action: "refresh", view: view, err: err}
	}
}

func (b *savedBrowser) selectedID() string {
	if len(b.views) == 0 {
		return ""
	}
	return b.views[b.selected].Reference.ID
}

func (b *savedBrowser) beginInput(state string) tea.Cmd {
	b.returnState = b.state
	if b.state == "details" {
		b.targetID = b.current.Reference.ID
	} else {
		b.targetID = b.selectedID()
	}
	b.state = state
	b.input.SetValue("")
	b.err = ""
	return b.input.Focus()
}

func (b *savedBrowser) update(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if b.busy {
		if msg.String() == "esc" {
			return nil, true
		}
		return nil, false
	}
	key := msg.String()
	if key == "ctrl+c" {
		return tea.Quit, false
	}
	if b.state == "register" || b.state == "relocate" {
		switch key {
		case "esc":
			b.input.Blur()
			b.state = b.returnState
			return nil, false
		case "enter":
			path := b.input.Value()
			saved := b.saved
			id := b.targetID
			action := b.state
			b.input.Blur()
			b.busy = true
			return func() tea.Msg {
				var err error
				if action == "register" {
					_, err = saved.Register(context.Background(), path)
				} else {
					err = saved.Relocate(context.Background(), id, path)
				}
				return savedResultMsg{action: action, err: err}
			}, false
		default:
			var cmd tea.Cmd
			b.input, cmd = b.input.Update(msg)
			return cmd, false
		}
	}
	if b.state == "forgetConfirm" {
		switch key {
		case "esc", "n":
			b.state = b.returnState
		case "y":
			id, saved := b.targetID, b.saved
			b.busy = true
			return func() tea.Msg { return savedResultMsg{action: "forget", err: saved.Forget(context.Background(), id)} }, false
		}
		return nil, false
	}
	if b.state == "lockConfirm" {
		switch key {
		case "esc", "n":
			b.state = "lockPreview"
		case "y":
			file, preview, service := b.current.Reference.ManifestPath, b.preview, b.service
			b.busy = true
			return func() tea.Msg {
				return savedResultMsg{action: "lockWrite", err: service.ApplyProjectLockPreview(file, preview)}
			}, false
		}
		return nil, false
	}
	if b.state == "applyConfirm" {
		switch key {
		case "esc", "n":
			b.state = "plan"
		case "y":
			file, plan, service := b.current.Reference.ManifestPath, b.plan, b.service
			b.busy = true
			return func() tea.Msg {
				result, err := service.ApplyProject(file, plan)
				return savedResultMsg{action: "apply", result: result, err: err}
			}, false
		}
		return nil, false
	}
	file, service := b.current.Reference.ManifestPath, b.service
	switch b.state {
	case "list":
		switch key {
		case "esc":
			return nil, true
		case "up", "k":
			if len(b.views) > 0 {
				b.selected = (b.selected - 1 + len(b.views)) % len(b.views)
			}
		case "down", "j":
			if len(b.views) > 0 {
				b.selected = (b.selected + 1) % len(b.views)
			}
		case "enter", "d":
			if id := b.selectedID(); id != "" {
				saved := b.saved
				b.busy = true
				return func() tea.Msg {
					view, err := saved.Open(context.Background(), id)
					return savedResultMsg{action: "open", view: view, err: err}
				}, false
			}
		case "r":
			return b.beginInput("register"), false
		case "m":
			if b.selectedID() != "" {
				return b.beginInput("relocate"), false
			}
		case "f":
			if b.selectedID() != "" {
				b.targetID = b.selectedID()
				b.returnState = "list"
				b.state = "forgetConfirm"
			}
		}
	case "details":
		if b.current.ManifestError != nil && (key == "p" || key == "l" || key == "o") {
			b.err = b.current.ManifestError.Error()
			return nil, false
		}
		switch key {
		case "esc":
			return b.load(), false
		case "up", "k":
			b.offset = max(0, b.offset-1)
		case "down", "j":
			b.offset++
		case "p":
			b.busy = true
			return func() tea.Msg {
				plan, err := service.PlanProject(file)
				return savedResultMsg{action: "plan", plan: plan, err: err}
			}, false
		case "l":
			b.state = "lock"
		case "o":
			b.busy = true
			return func() tea.Msg {
				return savedResultMsg{action: "openDirectory", err: openProjectDirectory(filepath.Dir(file))}
			}, false
		case "m":
			return b.beginInput("relocate"), false
		case "f":
			b.targetID = b.current.Reference.ID
			b.returnState = "details"
			b.state = "forgetConfirm"
		}
	case "plan":
		switch key {
		case "esc", "enter":
			b.state = "details"
		case "up", "k":
			b.offset = max(0, b.offset-1)
		case "down", "j":
			b.offset++
		case "g":
			if !b.plan.Executable {
				b.err = "Plan is not executable"
				break
			}
			if err := servicePreflight(b.service, b.current.Reference.ManifestPath, b.plan); err != nil {
				b.err = err.Error()
				break
			}
			b.err = ""
			b.state = "applyConfirm"
		}
	case "lock":
		switch key {
		case "esc":
			b.state = "details"
		case "up", "k":
			b.offset = max(0, b.offset-1)
		case "down", "j":
			b.offset++
		case "v":
			b.busy = true
			return func() tea.Msg {
				_, err := service.LockProject(file, true, false)
				return savedResultMsg{action: "check", err: err}
			}, false
		case "c", "u":
			if key == "c" && b.current.LockFound {
				b.err = "Lock already exists; use u to preview an update"
				break
			}
			b.busy = true
			return func() tea.Msg {
				preview, err := service.PreviewProjectLock(file)
				return savedResultMsg{action: "preview", preview: preview, err: err}
			}, false
		case "r":
			b.busy = true
			return func() tea.Msg {
				return savedResultMsg{action: "restore", err: service.RestoreLockedProject(context.Background(), file)}
			}, false
		}
	case "lockPreview":
		switch key {
		case "esc":
			b.state = "lock"
		case "enter":
			b.state = "lockConfirm"
		case "up", "k":
			b.offset = max(0, b.offset-1)
		case "down", "j":
			b.offset++
		}
	}
	return nil, false
}

func servicePreflight(service application.ProjectService, file string, plan planning.Plan) error {
	return service.PreflightProject(file, plan)
}

func openProjectDirectory(directory string) error {
	command := "xdg-open"
	if runtime.GOOS == "windows" {
		command = "explorer"
	} else if runtime.GOOS == "darwin" {
		command = "open"
	}
	process := exec.Command(command, directory)
	if err := process.Start(); err != nil {
		return err
	}
	return process.Process.Release()
}

func (b *savedBrowser) view(width, height int) string {
	lines := []string{"ProvisionHub · My Projects", ""}
	switch b.state {
	case "list", "forgetConfirm", "register", "relocate":
		if b.state == "register" || b.state == "relocate" {
			label := "Register Existing Project"
			if b.state == "relocate" {
				label = "Relocate Project"
			}
			lines = append(lines, label, "Manifest path or project directory:", b.input.View(), "enter save · esc cancel")
		} else {
			if len(b.views) == 0 {
				lines = append(lines, "No saved projects yet.")
			}
			for i, view := range b.views {
				prefix := "  "
				if i == b.selected {
					prefix = "> "
				}
				name, state := view.Manifest.Metadata.Name, "available"
				if view.ManifestError != nil {
					name, state = "(unavailable)", view.ManifestError.Error()
				}
				lines = append(lines, fmt.Sprintf("%s%s · %d components · %s", prefix, name, len(view.Manifest.CanonicalComponents()), state), "    "+view.Reference.ManifestPath)
			}
			lines = append(lines, "", "enter open/details · r register · m relocate · f forget · esc home")
			if b.state == "forgetConfirm" {
				lines = append(lines, "Forget Project? Only the SQLite reference is removed.", "y confirm · n/esc cancel")
			}
		}
	case "details":
		view := b.current
		if view.ManifestError != nil {
			lines = append(lines, "Unavailable project", "Manifest: "+view.Reference.ManifestPath, "Error: "+view.ManifestError.Error(), "", "m relocate · f forget · esc list")
			break
		}
		lines = append(lines, view.Manifest.Metadata.Name, "Manifest: "+view.Reference.ManifestPath, "Components:")
		if view.Reference.LastOpenedAt != nil {
			lines = append(lines, "Last opened: "+view.Reference.LastOpenedAt.Local().Format("2006-01-02 15:04"))
		}
		for _, component := range view.Manifest.CanonicalComponents() {
			lines = append(lines, fmt.Sprintf("  %s  %s  %s", component.Name, component.Type, b.service.TechnologyName(component.Technology)))
		}
		if len(view.Manifest.Spec.Connections) > 0 {
			lines = append(lines, "Connections:")
			for _, connection := range view.Manifest.CanonicalConnections() {
				lines = append(lines, fmt.Sprintf("  %s → %s (%s)", connection.From, connection.To, connection.Type))
			}
		}
		lines = append(lines, "Plugin Lock:")
		if view.LockError != nil {
			lines = append(lines, "  Invalid: "+view.LockError.Error())
		} else if !view.LockFound {
			lines = append(lines, "  Missing (unlocked)")
		} else {
			for _, plugin := range view.Lock.Plugins {
				lines = append(lines, fmt.Sprintf("  %s@%s (%s)", plugin.ID, plugin.Version, plugin.Source))
			}
		}
		lines = append(lines, "", "p view plan · l manage lock · o open directory · m relocate · f forget · esc list")
	case "plan", "applyConfirm":
		w := wizard{plan: b.plan, err: b.err, planOffset: b.offset, width: width, height: height, step: resultStep, confirm: b.state == "applyConfirm"}
		return w.planView(width, height)
	case "lock":
		lines = append(lines, "Plugin Lock · "+b.current.Manifest.Metadata.Name)
		if b.current.LockFound {
			for _, plugin := range b.current.Lock.Plugins {
				lines = append(lines, fmt.Sprintf("  %s@%s · %s · %s", plugin.ID, plugin.Version, plugin.Source, plugin.ContentSHA256))
			}
		} else {
			lines = append(lines, "No lockfile")
		}
		lines = append(lines, "", "v validate · c create · u preview update · r restore official artifacts · esc back")
	case "lockPreview", "lockConfirm":
		lines = append(lines, "Plugin Lock Change Preview", "Current:")
		if !b.preview.Found {
			lines = append(lines, "  No lockfile")
		}
		for _, plugin := range b.preview.Existing.Plugins {
			lines = append(lines, fmt.Sprintf("  %s@%s · %s · %s", plugin.ID, plugin.Version, plugin.Source, plugin.ContentSHA256), "    "+plugin.Artifact)
		}
		lines = append(lines, "Proposed:")
		for _, plugin := range b.preview.Proposed.Plugins {
			lines = append(lines, fmt.Sprintf("  %s@%s · %s · %s", plugin.ID, plugin.Version, plugin.Source, plugin.ContentSHA256), "    "+plugin.Artifact)
		}
		if b.state == "lockConfirm" {
			lines = append(lines, "y confirm write · n/esc cancel")
		} else {
			lines = append(lines, "enter request confirmation · esc back")
		}
	}
	if b.busy {
		lines = append(lines, "", "Working…")
	}
	if b.notice != "" {
		lines = append(lines, "", b.notice)
	}
	if b.err != "" && b.state != "plan" && b.state != "applyConfirm" {
		lines = append(lines, "", "Error: "+b.err)
	}
	if height < 1 {
		height = 1
	}
	if width < 1 {
		width = 1
	}
	start := 0
	if len(lines) > height {
		start = max(0, min(b.offset, len(lines)-height))
		if b.state == "list" || b.state == "forgetConfirm" {
			selectedLine := 3 + 2*b.selected
			if selectedLine >= start+height {
				start = min(len(lines)-height, selectedLine-height+2)
			}
			if selectedLine < start {
				start = selectedLine
			}
		}
	}
	visible := lines[start:min(len(lines), start+height)]
	for i := range visible {
		visible[i] = fitPlanLine(visible[i], width)
	}
	return strings.Join(visible, "\n")
}
