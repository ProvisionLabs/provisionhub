package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func (w *wizard) planLines(width int) []string {
	lines := []string{
		"Project: " + w.plan.ProjectName,
		"Manifest: " + w.plan.ManifestVersion,
		"Status: " + string(w.plan.Status),
		"Execution: unavailable (no execution engine)",
		"", "Selected Technologies:",
	}
	for _, technology := range w.plan.Technologies {
		lines = append(lines, fmt.Sprintf("  %s: %s (%s)", technology.Category, technology.Name, technology.ID))
	}
	lines = append(lines, "", "Planned Capabilities:")
	for _, operation := range w.plan.Operations {
		lines = append(lines, fmt.Sprintf("  %s: %s [%s]", operation.ID, operation.Description, operation.Status))
		if len(operation.Dependencies) != 0 {
			lines = append(lines, "    Depends on: "+strings.Join(operation.Dependencies, ", "))
		}
	}
	lines = append(lines, "", "Diagnostics:")
	for _, diagnostic := range w.plan.Diagnostics {
		lines = append(lines, fmt.Sprintf("  %s [%s]: %s", diagnostic.Severity, diagnostic.Code, diagnostic.Message))
	}
	wrapped := []string{}
	for _, line := range lines {
		current := ""
		for _, r := range line {
			part := string(r)
			if lipgloss.Width(part) > width {
				part = "?"
			}
			if lipgloss.Width(current+part) > width {
				wrapped = append(wrapped, current)
				current = ""
			}
			current += part
		}
		wrapped = append(wrapped, current)
	}
	return wrapped
}

func planPageHeight(height int) int { return max(1, height-2) }

func (w *wizard) clampPlanOffset() {
	last := max(0, len(w.planLines(max(1, w.width)))-planPageHeight(w.height))
	w.planOffset = max(0, min(w.planOffset, last))
}

func (w *wizard) planView(width, height int) string {
	content := w.planLines(width)
	capacity := planPageHeight(height)
	offset := max(0, min(w.planOffset, len(content)-capacity))
	lines := []string{}
	if height >= 3 {
		lines = append(lines, fitPlanLine("ProvisionHub — Generation Plan", width))
	}
	lines = append(lines, content[offset:min(len(content), offset+capacity)]...)
	if height >= 2 {
		footer := fmt.Sprintf("↑↓/jk scroll · enter/esc review · q quit (%d/%d)", offset+1, len(content))
		lines = append(lines, fitPlanLine(footer, width))
	}
	return strings.Join(lines, "\n")
}

func fitPlanLine(text string, width int) string {
	result := ""
	for _, r := range text {
		if lipgloss.Width(result+string(r)) > width {
			break
		}
		result += string(r)
	}
	return result
}
