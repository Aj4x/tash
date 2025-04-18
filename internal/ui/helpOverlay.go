package ui

import "github.com/charmbracelet/lipgloss"

// RenderHelpOverlay renders an overlay with all available commands
func RenderHelpOverlay(m *Model) string {
	// Calculate overlay dimensions
	overlayWidth := int(float64(m.Width) * 0.7)

	// Get the viewport content
	helpContent := m.HelpViewport.View()

	// Add scroll indicators if needed
	scrollIndicator := ""
	if !m.HelpViewport.AtBottom() {
		scrollIndicator = "\n↓ Scroll for more"
	}
	if !m.HelpViewport.AtTop() {
		scrollIndicator = "↑ More above\n" + scrollIndicator
	}

	if scrollIndicator != "" {
		scrollStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
		helpContent = helpContent + "\n" + scrollStyle.Render(scrollIndicator)
	}

	// Wrap the content in the overlay style
	overlay := GeneralOverlayStyle(overlayWidth).Render(helpContent)

	return lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
}
