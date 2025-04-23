package ui

import (
	"github.com/Aj4x/tash/internal/task"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

// RenderTaskDetailOverlay renders an overlay with detailed task information
func RenderTaskDetailOverlay(width, height int, selectedTask *task.Task) string {
	if selectedTask == nil {
		return ""
	}

	// Calculate overlay dimensions
	overlayWidth := int(float64(width) * 0.7)
	overlayHeight := int(float64(height) * 0.7)

	// Format aliases as a comma-separated list
	aliases := strings.Join(selectedTask.Aliases, ", ")

	// Build the content
	content := TaskDetailOverlayTitleStyle.Render("Task Details") + "\n\n"
	content += TaskDetailOverlayLabelStyle.Render("ID: ") + selectedTask.Id + "\n\n"
	content += TaskDetailOverlayLabelStyle.Render("Summary: ") + selectedTask.Summary + "\n\n"
	content += TaskDetailOverlayLabelStyle.Render("Description: ") + selectedTask.Desc + "\n\n"
	content += TaskDetailOverlayLabelStyle.Render("Aliases: ") + aliases + "\n"

	// Wrap the content in the overlay style
	overlay := TaskDetailOverlayStyle(overlayWidth, overlayHeight).Render(content)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
}
