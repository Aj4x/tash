package ui

import (
	"encoding/json"
	"fmt"
	"github.com/Aj4x/tash/internal/task"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"os/exec"
	"slices"
	"strings"
)

// Control represents a UI control that can be focused
type Control int

const (
	ControlTable Control = iota
	ControlViewport

	// put any new controls above this line, so our ControlMax, used for tabbing
	// though the controls, stays the maximum Control value
	ControlMax
)

// Tab cycles to the next control
func (c Control) Tab() Control {
	tabbedControl := Control(int(c) + 1)
	if tabbedControl >= ControlMax {
		return Control(0)
	}
	return tabbedControl
}

// TextWrap wraps text to fit within a specified width
func TextWrap(s string, n int) []string {
	if n <= 0 {
		return nil
	}
	var lines []string
	remaining := s
	for runewidth.StringWidth(remaining) >= n {
		lines = append(lines, remaining[:n])
		remaining = remaining[n:]
	}
	lines = append(lines, remaining)
	return lines
}

// RenderHelpView renders the help text at the bottom of the screen
func RenderHelpView(taskRunning bool, showTaskPicker bool, hasSelectedTasks bool) string {
	// If task picker is shown, show picker-specific help
	if showTaskPicker {
		help := []string{
			"esc: close picker",
			"↑/↓: navigate matches",
			"tab: autocomplete",
			"enter: select task",
		}
		return HelpStyle.Render(strings.Join(help, " • "))
	}

	// For normal view, show a more concise help text with the most important commands
	help := []string{
		"q: quit",
		"tab: switch focus",
		"↑/↓/j/k: navigate",
		"enter/e: execute task",
		"/: task picker",
		"?: help",
	}

	// Add the Ctrl+x help text only when a task is running
	if taskRunning {
		help = append(help, "ctrl+x: cancel task")
	}

	// Add task selection related help text when there are selected tasks
	if hasSelectedTasks {
		help = append(help, "ctrl+e: execute selected tasks")
	}

	return HelpStyle.Render(strings.Join(help, " • "))
}

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

// Model represents the UI model for the application
type Model struct {
	Tasks              []task.Task `json:"-"`
	TasksLoading       bool
	Result             *string                  `json:"-"`
	TaskChan           chan string              `json:"-"`
	OutChan            chan string              `json:"-"`
	ErrChan            chan string              `json:"-"`
	CmdChan            chan task.TaskCommandMsg `json:"-"`
	Viewport           viewport.Model           `json:"-"`
	Table              table.Model              `json:"-"`
	Focused            Control
	Width              int
	Height             int
	Initialised        bool
	SelectedTask       *task.Task
	ShowDetailsOverlay bool
	ShowHelpOverlay    bool
	HelpViewport       viewport.Model `json:"-"` // Viewport for scrollable help content
	Command            *exec.Cmd      `json:"-"`
	TaskRunning        bool

	// Task picker fields
	ShowTaskPicker     bool
	TaskPickerInput    string
	TaskPickerMatches  []task.Task `json:"-"`
	TaskPickerSelected int

	// Selected tasks for batch execution
	SelectedTasks         []task.Task
	ExecutingBatch        bool
	CurrentBatchTaskIndex int
}

// NewModel creates a new UI model
func NewModel() Model {
	columns := []table.Column{
		{Title: "Id", Width: 30},
		{Title: "Description", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	t.SetStyles(table.Styles{
		Header:   TableHeaderStyle,
		Selected: TableSelectedStyle,
	})

	return Model{
		Tasks:              []task.Task{},
		Result:             new(string),
		TaskChan:           make(chan string),
		OutChan:            make(chan string),
		ErrChan:            make(chan string),
		CmdChan:            make(chan task.TaskCommandMsg),
		Viewport:           viewport.New(0, 0),
		Table:              t,
		Focused:            ControlTable,
		Initialised:        false,
		SelectedTask:       nil,
		ShowDetailsOverlay: false,
		ShowHelpOverlay:    false,
		HelpViewport:       viewport.New(0, 0),

		// Initialize task picker fields
		ShowTaskPicker:     false,
		TaskPickerInput:    "",
		TaskPickerMatches:  []task.Task{},
		TaskPickerSelected: 0,

		// Initialize selected tasks
		SelectedTasks: []task.Task{},
	}
}

// View renders the UI
func (m Model) View() string {
	if !m.Initialised {
		return "Initialising..."
	}

	tableRendered := m.Table.View()
	viewportRendered := m.Viewport.View()

	if m.Focused == ControlTable {
		tableRendered = FocusedStyle.Render(tableRendered)
		viewportRendered = ViewportStyle.Render(viewportRendered)
	} else if m.Focused == ControlViewport {
		tableRendered = TableStyle.Render(tableRendered)
		viewportRendered = FocusedStyle.Render(viewportRendered)
	}

	// Build the layout
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, tableRendered, viewportRendered)

	// Add selected tasks display if there are any
	var selectedTasksText string
	if len(m.SelectedTasks) > 0 {
		taskNames := make([]string, len(m.SelectedTasks))
		for i, t := range m.SelectedTasks {
			taskNames[i] = t.Id
		}
		selectedTasksText = TableSelectedTaskStyle.Render(
			fmt.Sprintf("Selected tasks (%d): %s",
				len(m.SelectedTasks),
				strings.Join(taskNames, ", ")),
		)
	}

	// Add help text at the bottom
	helpText := RenderHelpView(m.TaskRunning, m.ShowTaskPicker, len(m.SelectedTasks) > 0)

	// Combine everything
	var fullView string
	if len(m.SelectedTasks) > 0 {
		fullView = lipgloss.JoinVertical(lipgloss.Left, mainView, selectedTasksText, helpText)
	} else {
		fullView = lipgloss.JoinVertical(lipgloss.Left, mainView, helpText)
	}

	// If details overlay is visible, render it on top
	if m.ShowDetailsOverlay {
		return RenderTaskDetailOverlay(m.Width, m.Height, m.SelectedTask)
	}

	// If task picker is visible, render it on top
	if m.ShowTaskPicker {
		return RenderTaskPicker(m.Width, m.Height, m.TaskPickerInput, m.TaskPickerMatches, m.TaskPickerSelected)
	}

	// If help overlay is visible, render it on top
	if m.ShowHelpOverlay {
		return RenderHelpOverlay(&m)
	}

	return fullView
}

// AppendToViewport adds text to the viewport
func (m *Model) AppendToViewport(msg string, style lipgloss.Style) {
	lines := TextWrap(msg, m.Viewport.Width)
	for _, line := range lines {
		*m.Result += "\n" + style.Render(line)
	}
	m.Viewport.SetContent(*m.Result)
	m.Viewport.GotoBottom()
}

// AppendAppMsg adds an application message to the viewport
func (m *Model) AppendAppMsg(msg string) {
	m.AppendToViewport(msg, AppMsgStyle)
}

// AppendErrorMsg adds an error message to the viewport
func (m *Model) AppendErrorMsg(msg string) {
	m.AppendToViewport(msg, ErrorMsgStyle)
}

// AppendCommandOutput adds command output to the viewport
func (m *Model) AppendCommandOutput(msg string) {
	m.AppendToViewport(msg, OutputStyle)
}

// UpdateTaskTable updates the task table with the current tasks
func (m *Model) UpdateTaskTable() {
	var rows []table.Row
	for _, t := range m.Tasks {
		rows = append(rows, table.Row{
			t.Id,
			t.Desc,
		})
	}
	m.Table.SetRows(rows)
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return m.RefreshTaskList()
}

// Update handles messages and updates the model
// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case task.TaskCommandMsg:
		return m.handleTaskCommandMsg(msg)
	case task.TaskMsg:
		m.AppendCommandOutput(string(msg))
		cmds = append(cmds, m.WaitForTaskMsg())
		m.AppendTask(string(msg))
		slices.SortFunc(m.Tasks, func(a, b task.Task) int {
			return strings.Compare(a.Id, b.Id)
		})
	case task.TaskOutputMsg:
		m.AppendCommandOutput(string(msg))
		cmds = append(cmds, m.WaitForTaskOutputMsg())
	case task.TaskOutputErrMsg:
		m.AppendErrorMsg(string(msg))
		cmds = append(cmds, m.WaitForTaskErrorMsg())
	case task.TaskErrMsg:
		m.AppendErrorMsg(msg.Err.Error())
		if m.ExecutingBatch {
			m.AppendErrorMsg("Batch execution aborted")
			m.ExecutingBatch = false
			m.CurrentBatchTaskIndex = -1
		}
		cmds = append(cmds, m.WaitForTaskMsg())
	case task.ListAllErrMsg:
		m.TasksLoading = false
		m.AppendErrorMsg("Error: " + msg.Err.Error())
	case task.TaskDoneMsg:
		m.TasksLoading = false
		m.AppendAppMsg("Task executed successfully!\n")
		if m.ExecutingBatch {
			return m.executeNextSelectedTask(m.CurrentBatchTaskIndex)
		}
	case task.ListAllDoneMsg:
		m.TasksLoading = false
		m.AppendAppMsg("Task list refreshed successfully!\n")
		m.UpdateTaskTable()
	}

	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

// handleWindowSizeMsg handles window resize events
func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	if !m.Initialised {
		m.Initialised = true
	}
	m.HandleWindowResize(msg.Width, msg.Height)
	return m, nil
}

// handleKeyMsg processes all keyboard input
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle task picker keys first
	if m.ShowTaskPicker {
		return m.handleTaskPickerKey(msg)
	}

	// Handle details overlay keys next
	if m.ShowDetailsOverlay {
		if msg.String() == "esc" || msg.String() == "i" {
			m.ShowDetailsOverlay = false
		}
		return m, nil
	}

	// Handle help overlay keys
	if m.ShowHelpOverlay {
		switch msg.String() {
		case "esc", "?":
			m.ShowHelpOverlay = false
		case "up", "k":
			m.HelpViewport.LineUp(1)
		case "down", "j":
			m.HelpViewport.LineDown(1)
		case "pgup":
			m.HelpViewport.HalfViewUp()
		case "pgdown":
			m.HelpViewport.HalfViewDown()
		case "home":
			m.HelpViewport.GotoTop()
		case "end":
			m.HelpViewport.GotoBottom()
		}
		return m, nil
	}

	// Handle general keys
	switch msg.String() {
	case "q":
		return m, tea.Quit
	case "ctrl+l":
		if m.TasksLoading {
			return m, nil
		}
		m.Result = new(string)
		m.Viewport.SetContent(*m.Result)
		m.Viewport.GotoTop()
	case "ctrl+r":
		if m.TasksLoading {
			return m, nil
		}
		return m, m.RefreshTaskList()
	case "i":
		return m.handleInfoKey()
	case "tab":
		return m.handleTabKey()
	case "up", "down", "j", "k":
		return m.handleNavigationKey(msg)
	case "enter", "e":
		return m.handleExecuteKey()
	case "ctrl+x":
		return m.handleCancelTask()
	case "/":
		return m.handleOpenTaskPicker()
	case "ctrl+e":
		return m.handleExecuteSelectedTasks()
	case "ctrl+d":
		return m.handleClearSelectedTasks()
	case "?":
		return m.handleHelpKey()
	}

	return m, nil
}

// handleInfoKey shows task details when 'i' is pressed
func (m Model) handleInfoKey() (tea.Model, tea.Cmd) {
	if m.Focused == ControlTable && len(m.Tasks) > 0 && m.Table.SelectedRow() != nil {
		selectedIndex := m.Table.Cursor()
		m.SelectedTask = &m.Tasks[selectedIndex]
		m.ShowDetailsOverlay = true
	}
	return m, nil
}

// handleTabKey handles tab key to switch focus between controls
func (m Model) handleTabKey() (tea.Model, tea.Cmd) {
	m.Focused = m.Focused.Tab()
	if m.Focused == ControlTable {
		m.Table.Focus()
	} else {
		m.Table.Blur()
	}
	return m, nil
}

// handleNavigationKey handles navigation keys (up, down, j, k)
func (m Model) handleNavigationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch m.Focused {
	case ControlTable:
		m.Table, cmd = m.Table.Update(msg)
		cmds = append(cmds, cmd)
	case ControlViewport:
		m.Viewport, cmd = m.Viewport.Update(msg)
		cmds = append(cmds, cmd)
	default:
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

// handleExecuteKey executes the selected task when enter or 'e' is pressed
func (m Model) handleExecuteKey() (tea.Model, tea.Cmd) {
	if m.TasksLoading {
		return m, nil
	}

	if m.Focused == ControlTable && len(m.Tasks) > 0 && m.Table.SelectedRow() != nil {
		return m, m.ExecuteSelectedTask()
	}

	return m, nil
}

// handleCancelTask cancels a running task
func (m Model) handleCancelTask() (tea.Model, tea.Cmd) {
	if m.TaskRunning {
		if err := task.StopTaskProcess(m.Command.Process); err != nil {
			m.AppendErrorMsg("Error cancelling task: " + err.Error())
			return m, nil
		}
		m.TaskRunning = false
		m.Command = nil
		m.AppendAppMsg("Task cancelled\n")
	}
	return m, nil
}

// handleOpenTaskPicker opens the task picker
func (m Model) handleOpenTaskPicker() (tea.Model, tea.Cmd) {
	if m.TasksLoading {
		return m, nil
	}

	m.ShowTaskPicker = true
	m.TaskPickerInput = ""
	m.TaskPickerMatches = m.Tasks // Initialize with all tasks
	m.TaskPickerSelected = 0

	return m, nil
}

// handleTaskPickerKey handles key presses when the task picker is open
func (m Model) handleTaskPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Close the task picker
		m.ShowTaskPicker = false
		m.Focused = ControlTable
		return m, nil

	case "enter":
		// Select the current task
		if len(m.TaskPickerMatches) > 0 && m.TaskPickerSelected < len(m.TaskPickerMatches) {
			selectedTask := m.TaskPickerMatches[m.TaskPickerSelected]

			// Check if task is already in selected tasks
			alreadySelected := false
			for _, t := range m.SelectedTasks {
				if t.Id == selectedTask.Id {
					alreadySelected = true
					break
				}
			}

			// Add to selected tasks if not already there
			if !alreadySelected {
				m.SelectedTasks = append(m.SelectedTasks, selectedTask)
				m.AppendAppMsg(fmt.Sprintf("Added task '%s' to execution list\n", selectedTask.Id))
			}

			// Close the picker
			m.ShowTaskPicker = false
			m.Focused = ControlTable
		}
		return m, nil

	case "tab":
		// Autocomplete with the selected match
		if len(m.TaskPickerMatches) > 0 && m.TaskPickerSelected < len(m.TaskPickerMatches) {
			m.TaskPickerInput = m.TaskPickerMatches[m.TaskPickerSelected].Id
			// Update matches based on the new input
			m.updateTaskPickerMatches()
		}
		return m, nil

	case "up", "k":
		// Navigate up in matches
		if m.TaskPickerSelected > 0 {
			m.TaskPickerSelected--
		}
		return m, nil

	case "down", "j":
		// Navigate down in matches
		if m.TaskPickerSelected < len(m.TaskPickerMatches)-1 {
			m.TaskPickerSelected++
		}
		return m, nil
	}

	// Handle character input
	if len(msg.String()) == 1 || msg.String() == "backspace" {
		if msg.String() == "backspace" && len(m.TaskPickerInput) > 0 {
			// Remove last character
			m.TaskPickerInput = m.TaskPickerInput[:len(m.TaskPickerInput)-1]
		} else if msg.String() != "backspace" {
			// Add character to input
			m.TaskPickerInput += msg.String()
		}

		// Update matches based on the new input
		m.updateTaskPickerMatches()
		return m, nil
	}

	return m, nil
}

// updateTaskPickerMatches updates the task picker matches based on the current input
func (m *Model) updateTaskPickerMatches() {
	if m.TaskPickerInput == "" {
		m.TaskPickerMatches = m.Tasks
		return
	}

	// Filter tasks based on input
	var matches []task.Task
	input := strings.ToLower(m.TaskPickerInput)

	for _, t := range m.Tasks {
		// Check if input matches task ID
		if strings.Contains(strings.ToLower(t.Id), input) {
			matches = append(matches, t)
			continue
		}

		// Check if input matches any alias
		for _, alias := range t.Aliases {
			if strings.Contains(strings.ToLower(alias), input) {
				matches = append(matches, t)
				break
			}
		}
	}

	m.TaskPickerMatches = matches

	// Reset selected index if out of bounds
	if m.TaskPickerSelected >= len(matches) {
		m.TaskPickerSelected = 0
	}
}

// handleExecuteSelectedTasks executes all selected tasks
func (m Model) handleExecuteSelectedTasks() (tea.Model, tea.Cmd) {
	if m.TasksLoading || len(m.SelectedTasks) == 0 || m.ExecutingBatch {
		return m, nil
	}

	m.ExecutingBatch = true
	m.CurrentBatchTaskIndex = 0

	m.AppendAppMsg(fmt.Sprintf("Executing %d selected tasks\n", len(m.SelectedTasks)))

	// Execute the first task
	return m.executeNextSelectedTask(m.CurrentBatchTaskIndex)
}

// handleClearSelectedTasks clears the list of selected tasks
func (m Model) handleClearSelectedTasks() (tea.Model, tea.Cmd) {
	if len(m.SelectedTasks) > 0 {
		m.SelectedTasks = []task.Task{}
		m.AppendAppMsg("Selected tasks cleared\n")
	}
	return m, nil
}

// generateHelpContent creates the help content with a two-column layout
func generateHelpContent(overlayWidth int) string {
	// Calculate column width (accounting for padding and border)
	contentWidth := overlayWidth - 6      // 6 = 2*2 padding + 2 border
	columnWidth := (contentWidth / 2) - 2 // 2 for spacing between columns

	// Build the content with two columns
	content := HelpTextTitleStyle.Render("Help - Available Commands") + "\n\n"

	// Navigation commands
	content += HelpTextSectionStyle.Render("Navigation") + "\n"

	// Create two columns for navigation commands
	navCol1 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("tab: ") + "Switch focus\n" +
			HelpTextCommandStyle.Render("↑/↓/j/k: ") + "Navigate\n" +
			HelpTextCommandStyle.Render("q: ") + "Quit")

	navCol2 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("↑/↓: ") + "Scroll help\n" +
			HelpTextCommandStyle.Render("pgup/pgdn: ") + "Page up/down\n" +
			HelpTextCommandStyle.Render("home/end: ") + "Top/bottom")

	content += lipgloss.JoinHorizontal(lipgloss.Top, navCol1, "  ", navCol2) + "\n\n"

	// Task commands
	content += HelpTextSectionStyle.Render("Task Management") + "\n"

	taskCol1 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("enter/e: ") + "Execute task\n" +
			HelpTextCommandStyle.Render("i: ") + "Task details\n" +
			HelpTextCommandStyle.Render("ctrl+r: ") + "Refresh tasks")

	taskCol2 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("ctrl+x: ") + "Cancel task\n" +
			HelpTextCommandStyle.Render("ctrl+l: ") + "Clear output")

	content += lipgloss.JoinHorizontal(lipgloss.Top, taskCol1, "  ", taskCol2) + "\n\n"

	// Task picker commands
	content += HelpTextSectionStyle.Render("Task Picker") + "\n"

	pickerCol1 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("/: ") + "Open picker\n" +
			HelpTextCommandStyle.Render("tab: ") + "Autocomplete")

	pickerCol2 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("enter: ") + "Select task\n" +
			HelpTextCommandStyle.Render("esc: ") + "Close picker")

	content += lipgloss.JoinHorizontal(lipgloss.Top, pickerCol1, "  ", pickerCol2) + "\n\n"

	// Batch execution commands
	content += HelpTextSectionStyle.Render("Batch Execution") + "\n"

	batchCol1 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("ctrl+e: ") + "Execute tasks")

	batchCol2 := lipgloss.NewStyle().Width(columnWidth).Render(
		HelpTextCommandStyle.Render("ctrl+d: ") + "Clear tasks")

	content += lipgloss.JoinHorizontal(lipgloss.Top, batchCol1, "  ", batchCol2) + "\n\n"

	// Help commands
	content += HelpTextSectionStyle.Render("Help") + "\n"
	content += HelpTextCommandStyle.Render("?: ") + "Show/hide help"

	return content
}

// handleHelpKey toggles the help overlay
func (m Model) handleHelpKey() (tea.Model, tea.Cmd) {
	m.ShowHelpOverlay = !m.ShowHelpOverlay

	// If showing the overlay, initialize the viewport content and dimensions
	if m.ShowHelpOverlay {
		// Calculate overlay dimensions
		overlayWidth := int(float64(m.Width) * 0.7)
		overlayHeight := int(float64(m.Height) * 0.7)
		contentWidth := overlayWidth - 6    // 6 = 2*2 padding + 2 border
		viewportHeight := overlayHeight - 6 // Account for padding and borders

		// Set viewport dimensions
		m.HelpViewport.Width = contentWidth
		m.HelpViewport.Height = viewportHeight

		// Generate and set content
		content := generateHelpContent(overlayWidth)
		modelBytes, err := json.MarshalIndent(m, "", "\t")
		content += "\n\n" + HelpTextSectionStyle.Render("Model")
		if err != nil {
			content += "\n" + HelpTextCommandStyle.Render(err.Error())
		}
		for _, text := range TextWrap(string(modelBytes), contentWidth) {
			content += "\n" + HelpTextCommandStyle.Render(text)
		}
		m.HelpViewport.SetContent(content)
		m.HelpViewport.GotoTop()
	}

	return m, nil
}

// executeNextSelectedTask executes the task at the given index and then executes the next task
func (m Model) executeNextSelectedTask(index int) (tea.Model, tea.Cmd) {
	if index >= len(m.SelectedTasks) {
		// All tasks have been executed
		m.AppendAppMsg("All selected tasks have been executed\n")
		m.ExecutingBatch = false
		m.CurrentBatchTaskIndex = -1
		return m, nil
	}

	selectedTask := m.SelectedTasks[index]
	m.CurrentBatchTaskIndex++
	m.AppendAppMsg(fmt.Sprintf("Executing task %d/%d: %s\n\n", index+1, len(m.SelectedTasks), selectedTask.Id))
	m.TasksLoading = true

	// Create a command that will execute the current task and then execute the next task
	return m, tea.Batch(
		task.ExecuteTask(selectedTask.Id, m.OutChan, m.CmdChan, m.ErrChan),
		m.WaitForTaskCommandMsg(), m.WaitForTaskErrorMsg(), m.WaitForTaskOutputMsg(),
	)
}

// handleTaskCommandMsg processes task command messages
func (m Model) handleTaskCommandMsg(msg task.TaskCommandMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	m.TaskRunning = msg.TaskRunning
	m.Command = msg.Command
	cmds = append(cmds, m.WaitForTaskCommandMsg())
	return m, tea.Batch(cmds...)
}

// RefreshTaskList refreshes the task list
func (m *Model) RefreshTaskList() tea.Cmd {
	m.Tasks = []task.Task{}
	m.TasksLoading = true
	m.AppendAppMsg("\nRefreshing task list\n")
	return tea.Batch(task.ListAll(m.TaskChan), m.WaitForTaskMsg())
}

// WaitForTaskMsg waits for a task message
func (m Model) WaitForTaskMsg() tea.Cmd {
	return func() tea.Msg {
		return task.TaskMsg(<-m.TaskChan)
	}
}

// WaitForTaskOutputMsg waits for a task output message
func (m Model) WaitForTaskOutputMsg() tea.Cmd {
	return func() tea.Msg {
		return task.TaskOutputMsg(<-m.OutChan)
	}
}

// WaitForTaskErrorMsg waits for a task error message
func (m Model) WaitForTaskErrorMsg() tea.Cmd {
	return func() tea.Msg {
		return task.TaskOutputErrMsg(<-m.ErrChan)
	}
}

// WaitForTaskCommandMsg waits for a task command message
func (m Model) WaitForTaskCommandMsg() tea.Cmd {
	return func() tea.Msg {
		msg := <-m.CmdChan
		return msg
	}
}

// AppendTask appends a task to the task list
func (m *Model) AppendTask(taskMsg string) {
	t, ok := task.ParseTaskLine(taskMsg)
	if !ok {
		return
	}
	m.Tasks = append(m.Tasks, t)
}

// HandleWindowResize handles window resize events
func (m *Model) HandleWindowResize(width, height int) {
	m.Width = width
	m.Height = height

	tableWidth := int(float64(m.Width) * 0.4)
	viewportWidth := m.Width - tableWidth - 4

	m.Table.SetWidth(tableWidth)
	m.Table.SetHeight(m.Height - 4)

	m.Viewport.Width = viewportWidth
	m.Viewport.Height = m.Height - 4
	m.Viewport.HighPerformanceRendering = false

	// Resize help viewport if needed
	if m.ShowHelpOverlay {
		overlayWidth := int(float64(m.Width) * 0.7)
		overlayHeight := int(float64(m.Height) * 0.7)
		contentWidth := overlayWidth - 6    // 6 = 2*2 padding + 2 border
		viewportHeight := overlayHeight - 6 // Account for padding and borders

		m.HelpViewport.Width = contentWidth
		m.HelpViewport.Height = viewportHeight
	}
}

// ExecuteSelectedTask executes the selected task
func (m *Model) ExecuteSelectedTask() tea.Cmd {
	if m.TasksLoading || len(m.Tasks) == 0 || m.Table.SelectedRow() == nil {
		return nil
	}

	selectedIndex := m.Table.Cursor()
	selectedTask := m.Tasks[selectedIndex]
	m.AppendAppMsg(fmt.Sprintf("Executing task: %s\n\n", selectedTask.Id))
	m.TasksLoading = true

	return tea.Batch(
		task.ExecuteTask(selectedTask.Id, m.OutChan, m.CmdChan, m.ErrChan),
		m.WaitForTaskCommandMsg(),
		m.WaitForTaskOutputMsg(),
		m.WaitForTaskErrorMsg(),
	)
}
