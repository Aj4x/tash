package ui

import (
	"fmt"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeyBindingSection represents a section of key bindings in the help text
type KeyBindingSection struct {
	Name        string       // Section name (e.g., "Navigation", "Task Management")
	KeyBindings []KeyBinding // Key bindings in this section
}

// Context is a type alias for string representing UI contexts where key bindings are active
type Context string

// Context constants
const (
	ContextGlobal         Context = "global"
	ContextTaskPicker     Context = "taskPicker"
	ContextHelpOverlay    Context = "helpOverlay"
	ContextDetailsOverlay Context = "detailsOverlay"
	ContextViewport       Context = "viewport"
)

// KeyBinding represents a single key binding with its key, description, and context
type KeyBinding struct {
	Key         string    // The key or key combination (e.g., "ctrl+c", "enter")
	Description string    // Description of what the key does
	Contexts    []Context // Contexts where this key binding is active
}

// KeyBindings contains all key bindings used in the application
type KeyBindings struct {
	Sections []KeyBindingSection // Sections of key bindings
}

// DefaultKeyBindings returns the default key bindings for the application
func DefaultKeyBindings() KeyBindings {
	return KeyBindings{
		Sections: []KeyBindingSection{
			{
				Name: "Help",
				KeyBindings: []KeyBinding{
					{Key: "?", Description: "Show/hide help", Contexts: []Context{ContextGlobal}},
				},
			},
			{
				Name: "Navigation",
				KeyBindings: []KeyBinding{
					{Key: "q", Description: "Quit", Contexts: []Context{ContextGlobal}},
					{Key: "tab", Description: "Switch focus", Contexts: []Context{ContextGlobal}},
					{Key: "↑/↓/j/k", Description: "Navigate", Contexts: []Context{ContextGlobal}},
					{Key: "pgup/pgdn", Description: "Page up/down", Contexts: []Context{ContextHelpOverlay, ContextViewport}},
					{Key: "home/end", Description: "Top/bottom", Contexts: []Context{ContextHelpOverlay, ContextViewport}},
				},
			},
			{
				Name: "Task Management",
				KeyBindings: []KeyBinding{
					{Key: "enter/e", Description: "Execute task", Contexts: []Context{ContextGlobal}},
					{Key: "i", Description: "Task details", Contexts: []Context{ContextGlobal}},
					{Key: "ctrl+r", Description: "Refresh tasks", Contexts: []Context{ContextGlobal}},
					{Key: "ctrl+x", Description: "Cancel task", Contexts: []Context{ContextGlobal}},
					{Key: "ctrl+l", Description: "Clear output", Contexts: []Context{ContextGlobal}},
				},
			},
			{
				Name: "Task Picker",
				KeyBindings: []KeyBinding{
					{Key: "/", Description: "Open picker", Contexts: []Context{ContextGlobal}},
					{Key: "tab", Description: "Autocomplete", Contexts: []Context{ContextTaskPicker}},
					{Key: "enter", Description: "Select task", Contexts: []Context{ContextTaskPicker}},
					{Key: "esc", Description: "Close picker", Contexts: []Context{ContextTaskPicker}},
					{Key: "↑/↓", Description: "Navigate matches", Contexts: []Context{ContextTaskPicker}},
				},
			},
			{
				Name: "Batch Execution",
				KeyBindings: []KeyBinding{
					{Key: "ctrl+e", Description: "Execute tasks", Contexts: []Context{ContextGlobal}},
					{Key: "ctrl+d", Description: "Clear tasks", Contexts: []Context{ContextGlobal}},
				},
			},
			{
				Name: "Details Overlay",
				KeyBindings: []KeyBinding{
					{Key: "esc/i", Description: "Close details", Contexts: []Context{ContextDetailsOverlay}},
				},
			},
		},
	}
}

// GetKeyBindingsForContext returns all key bindings for a specific context
func (kb KeyBindings) GetKeyBindingsForContext(context Context) []KeyBinding {
	var bindings []KeyBinding
	for _, section := range kb.Sections {
		for _, binding := range section.KeyBindings {
			for _, ctx := range binding.Contexts {
				if ctx == context {
					bindings = append(bindings, binding)
					break
				}
			}
		}
	}
	return bindings
}

// GetKeyBindingsForTaskPicker returns key bindings specific to the task picker
func (kb KeyBindings) GetKeyBindingsForTaskPicker() []KeyBinding {
	return kb.GetKeyBindingsForContext(ContextTaskPicker)
}

// GetKeyBindingsForGlobal returns key bindings for the global context
func (kb KeyBindings) GetKeyBindingsForGlobal() []KeyBinding {
	return kb.GetKeyBindingsForContext(ContextGlobal)
}

// GetKeyBindingsForHelpOverlay returns key bindings for the help overlay
func (kb KeyBindings) GetKeyBindingsForHelpOverlay() []KeyBinding {
	return kb.GetKeyBindingsForContext(ContextHelpOverlay)
}

// GetKeyBindingsForDetailsOverlay returns key bindings for the details overlay
func (kb KeyBindings) GetKeyBindingsForDetailsOverlay() []KeyBinding {
	return kb.GetKeyBindingsForContext(ContextDetailsOverlay)
}

// RenderHelpView renders the help text at the bottom of the screen using the key bindings
func (kb KeyBindings) RenderHelpView(taskRunning bool, showTaskPicker bool, hasSelectedTasks bool) string {
	// If task picker is shown, show picker-specific help
	if showTaskPicker {
		bindings := kb.GetKeyBindingsForTaskPicker()
		var help []string
		for _, binding := range bindings {
			help = append(help, fmt.Sprintf("%s: %s", binding.Key, binding.Description))
		}
		return HelpStyle.Render(strings.Join(help, " • "))
	}

	// For normal view, show a more concise help text with the most important commands
	bindings := kb.GetKeyBindingsForGlobal()
	var help []string

	// Add basic help items
	for _, binding := range bindings {
		// Skip task cancellation if no task is running
		if binding.Key == "ctrl+x" && !taskRunning {
			continue
		}

		// Skip batch execution if no tasks are selected
		if binding.Key == "ctrl+e" && !hasSelectedTasks {
			continue
		}

		help = append(help, fmt.Sprintf("%s: %s", binding.Key, binding.Description))
	}

	return HelpStyle.Render(strings.Join(help, " • "))
}

// GenerateHelpContent creates the help content with a two-column layout using the key bindings
func (kb KeyBindings) GenerateHelpContent(overlayWidth int) string {
	// Calculate column width (accounting for padding and border)
	contentWidth := overlayWidth - 6      // 6 = 2*2 padding + 2 border
	columnWidth := (contentWidth / 2) - 2 // 2 for spacing between columns

	// Build the content with two columns
	content := HelpTextTitleStyle.Render("Help - Available Commands")

	bi, ok := debug.ReadBuildInfo()
	if ok {
		version := bi.Main.Version
		content += HelpStyle.Render("\n" + version)
	}

	content += "\n\n"

	// Add each section
	for _, section := range kb.Sections {
		content += HelpTextSectionStyle.Render(section.Name) + "\n"

		// Split bindings into two columns
		bindings := section.KeyBindings
		midpoint := (len(bindings) + 1) / 2
		col1Bindings := bindings[:midpoint]
		col2Bindings := bindings[midpoint:]

		// Render column 1
		col1Content := ""
		for _, binding := range col1Bindings {
			col1Content += HelpTextCommandStyle.Render(binding.Key+": ") + binding.Description + "\n"
		}
		col1 := lipgloss.NewStyle().Width(columnWidth).Render(col1Content)

		// Render column 2
		col2Content := ""
		for _, binding := range col2Bindings {
			col2Content += HelpTextCommandStyle.Render(binding.Key+": ") + binding.Description + "\n"
		}
		col2 := lipgloss.NewStyle().Width(columnWidth).Render(col2Content)

		// Join columns
		content += lipgloss.JoinHorizontal(lipgloss.Top, col1, "  ", col2) + "\n\n"

		// output model for debugging

	}

	return content
}

// IsKeyMatch checks if a key message matches a key binding
func IsKeyMatch(msg tea.KeyMsg, keyBinding string) bool {
	// Handle special cases for key combinations
	switch keyBinding {
	case "enter/e":
		return msg.String() == "enter" || msg.String() == "e"
	case "↑/↓/j/k":
		return msg.String() == "up" || msg.String() == "down" || msg.String() == "j" || msg.String() == "k"
	case "pgup/pgdn":
		return msg.String() == "pgup" || msg.String() == "pgdown"
	case "home/end":
		return msg.String() == "home" || msg.String() == "end"
	case "esc/i":
		return msg.String() == "esc" || msg.String() == "i"
	default:
		return msg.String() == keyBinding
	}
}
