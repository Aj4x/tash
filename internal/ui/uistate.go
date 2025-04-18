package ui

// UIState represents the different states of the UI
type UIState int

const (
	// StateNormal is the default state of the UI
	StateNormal UIState = iota

	// StateTaskPicker is the state when the task picker is active
	StateTaskPicker

	// StateDetailsOverlay is the state when the task details overlay is active
	StateDetailsOverlay

	// StateHelpOverlay is the state when the help overlay is active
	StateHelpOverlay
)

// String returns a string representation of the UIState
func (s UIState) String() string {
	switch s {
	case StateNormal:
		return "Normal"
	case StateTaskPicker:
		return "TaskPicker"
	case StateDetailsOverlay:
		return "DetailsOverlay"
	case StateHelpOverlay:
		return "HelpOverlay"
	default:
		return "Unknown"
	}
}
