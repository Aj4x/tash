package ui

import "github.com/charmbracelet/lipgloss"

// Table Styles
var (
	TableStyle             = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	TableHeaderStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	TableSelectedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	TableSelectedTaskStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).PaddingLeft(1)
)

var (
	ViewportStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	FocusedStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("69"))
	HelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Message Styles
	AppMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // Green for app messages
	ErrorMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // Red for error messages
	OutputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))             // Default color for regular output
)

func GeneralOverlayStyle(overlayWidth int) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(overlayWidth)
}

// Task Picker styles
var (
	TaskPickerTitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		MarginBottom(1)
)

func TaskPickerInputStyle(overlayWidth int) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("241")).
		Padding(0, 1).
		Width(overlayWidth - 6)
}

func TaskPickerMatchStyle(overlayWidth int) lipgloss.Style {
	return lipgloss.NewStyle().
		Padding(0, 1).
		Width(overlayWidth - 6)
}

func TaskPickerSelectedMatchStyle(overlayWidth int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("63")).
		Bold(true).
		Padding(0, 1).
		Width(overlayWidth - 6)
}

// Task Detail Overlay styles
var (
	TaskDetailOverlayTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("63")).
					MarginBottom(1)
	TaskDetailOverlayLabelStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241"))
)

func TaskDetailOverlayStyle(overlayWidth, overlayHeight int) lipgloss.Style {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(overlayWidth).
		Height(overlayHeight)
}

// Help Text styles
var (
	HelpTextTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")).MarginBottom(1)
	HelpTextSectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).MarginTop(1).MarginBottom(1)
	HelpTextCommandStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("241"))
)
