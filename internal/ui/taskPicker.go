package ui

import (
	"github.com/Aj4x/tash/internal/task"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

// RenderTaskPicker renders the task picker overlay
func RenderTaskPicker(width, height int, input string, matches []task.Task, selectedIndex int) string {
	// Calculate overlay dimensions
	overlayWidth := int(float64(width) * 0.7)

	// Build the content
	content := TaskPickerTitleStyle.Render("Task Picker") + "\n\n"
	content += "Search: " + TaskPickerInputStyle(overlayWidth).Render(input) + "\n\n"

	if len(matches) > 0 {
		content += "Matching Tasks:\n"
		for i, match := range matches {
			taskText := match.Id
			if len(match.Aliases) > 0 {
				taskText += " (aliases: " + strings.Join(match.Aliases, ", ") + ")"
			}

			if i == selectedIndex {
				content += TaskPickerSelectedMatchStyle(overlayWidth).Render(taskText) + "\n"
			} else {
				content += TaskPickerMatchStyle(overlayWidth).Render(taskText) + "\n"
			}
		}
	} else if input != "" {
		content += "No matches found"
	}

	// Wrap the content in the overlay style
	overlay := GeneralOverlayStyle(overlayWidth).Render(content)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
}
