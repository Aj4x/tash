package ui

import (
	"encoding/json"
	"fmt"
	"github.com/Aj4x/tash/internal/msgbus"
	"github.com/Aj4x/tash/internal/task"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"os/exec"
	"strings"
	"time"
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
	for len(remaining) >= n {
		lines = append(lines, remaining[:n])
		remaining = remaining[n:]
	}
	lines = append(lines, remaining)
	return lines
}

// Model represents the UI model for the application
type Model struct {
	MessageBus   msgbus.PublisherSubscriber[task.Message] `json:"-"`
	busHandler   msgbus.MessageHandler[task.Message]
	Tasks        []task.Task `json:"-"`
	TasksLoading bool
	Result       *string        `json:"-"`
	Viewport     viewport.Model `json:"-"`
	Table        table.Model    `json:"-"`
	Focused      Control
	Width        int
	Height       int
	Initialised  bool
	SelectedTask *task.Task
	State        UIState        // Current UI state (normal, task picker, details overlay, help overlay)
	HelpViewport viewport.Model `json:"-"` // Viewport for scrollable help content
	Command      *exec.Cmd      `json:"-"`
	TaskRunning  bool
	KeyBindings  KeyBindings `json:"-"` // Key bindings for the application

	// Task picker fields
	TaskPickerInput    string
	TaskPickerMatches  []task.Task `json:"-"`
	TaskPickerSelected int

	// Selected tasks for batch execution
	SelectedTasks         []task.Task
	ExecutingBatch        bool
	CurrentBatchTaskIndex int
}

// NewModel creates a new UI model
func NewModel(bus msgbus.PublisherSubscriber[task.Message]) Model {
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
		MessageBus:   bus,
		busHandler:   make(msgbus.MessageHandler[task.Message], 4096),
		Tasks:        []task.Task{},
		Result:       new(string),
		Viewport:     viewport.New(0, 0),
		Table:        t,
		Focused:      ControlTable,
		Initialised:  false,
		SelectedTask: nil,
		State:        StateNormal,
		HelpViewport: viewport.New(0, 0),
		KeyBindings:  DefaultKeyBindings(),

		// Initialize task picker fields
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
	helpText := m.KeyBindings.RenderHelpView(m.TaskRunning, m.State == StateTaskPicker, len(m.SelectedTasks) > 0)

	// Combine everything
	var fullView string
	if len(m.SelectedTasks) > 0 {
		fullView = lipgloss.JoinVertical(lipgloss.Left, mainView, selectedTasksText, helpText)
	} else {
		fullView = lipgloss.JoinVertical(lipgloss.Left, mainView, helpText)
	}

	// Render the appropriate view based on the current state
	switch m.State {
	case StateDetailsOverlay:
		return RenderTaskDetailOverlay(m.Width, m.Height, m.SelectedTask)
	case StateTaskPicker:
		return RenderTaskPicker(m.Width, m.Height, m.TaskPickerInput, m.TaskPickerMatches, m.TaskPickerSelected)
	case StateHelpOverlay:
		return RenderHelpOverlay(&m)
	default: // StateNormal
		return fullView
	}
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
	sub := func(topic msgbus.Topic) {
		_, err := m.MessageBus.Subscribe(topic, m.busHandler)
		if err != nil {
			panic(fmt.Errorf("failed to subscribe to '%s' topic: %w", topic, err))
		}
	}
	topics := []msgbus.Topic{
		task.TopicTaskOutput,
		task.TopicTaskError,
		task.TopicTaskJSON,
		task.TopicTaskCommand,
		task.TopicTaskDone,
		task.TopicTaskListAllDone,
		task.TopicTaskListAllErr,
	}
	for _, t := range topics {
		sub(t)
	}
	return tea.Batch(
		m.RefreshTaskList(),
		m.pollMessages(),
	)
}

func parseTasksJson(jsonStr string) ([]task.Task, error) {
	var t struct {
		Tasks []task.Task `json:"tasks"`
	}
	err := json.Unmarshal([]byte(jsonStr), &t)
	if err != nil {
		return nil, err
	}
	return t.Tasks, nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case TickMessage:
		return m, m.pollMessages()

	// handle any bus messages
	case task.Message:
		// Process the message and set up another listener
		newModel, cmd := m.handleBusMessage(msg)
		if cmd == nil {
			return newModel, newModel.pollMessages()
		}
		return newModel, tea.Batch(cmd, newModel.pollMessages())
	case msgbus.TopicMessage[task.Message]:
		// Process the message and set up another listener
		newModel, cmd := m.handleBusMessage(msg.Message)
		if cmd == nil {
			return newModel, newModel.pollMessages()
		}
		return newModel, tea.Batch(cmd, newModel.pollMessages())
	default:
		return m, m.pollMessages()
	}
}

type TickMessage struct{}

func (m Model) pollMessages() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		select {
		case msg := <-m.busHandler:
			return msg.Message
		default:
			return TickMessage{}
		}
	})
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
	// Use a state machine approach to handle different UI states
	switch m.State {
	case StateTaskPicker:
		return m.handleTaskPickerKey(msg)
	case StateDetailsOverlay:
		return m.handleDetailsOverlayKey(msg)
	case StateHelpOverlay:
		return m.handleHelpOverlayKey(msg)
	default: // StateNormal
		return m.handleNormalKey(msg)
	}
}

// handleTaskPickerKey handles key presses when the task picker is open
func (m Model) handleTaskPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Close the task picker
	if IsKeyMatch(msg, "esc") {
		m.State = StateNormal
		m.Focused = ControlTable
		return m, nil
	}

	// Select the current task
	if IsKeyMatch(msg, "enter") {
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
			m.State = StateNormal
			m.Focused = ControlTable
		}
		return m, nil
	}

	// Autocomplete with the selected match
	if IsKeyMatch(msg, "tab") {
		if len(m.TaskPickerMatches) > 0 && m.TaskPickerSelected < len(m.TaskPickerMatches) {
			m.TaskPickerInput = m.TaskPickerMatches[m.TaskPickerSelected].Id
			// Update matches based on the new input
			m.updateTaskPickerMatches()
		}
		return m, nil
	}

	// Navigate up in matches
	if IsKeyMatch(msg, "up") || IsKeyMatch(msg, "k") {
		if m.TaskPickerSelected > 0 {
			m.TaskPickerSelected--
		}
		return m, nil
	}

	// Navigate down in matches
	if IsKeyMatch(msg, "down") || IsKeyMatch(msg, "j") {
		if m.TaskPickerSelected < len(m.TaskPickerMatches)-1 {
			m.TaskPickerSelected++
		}
		return m, nil
	}

	// Handle character input
	if len(msg.String()) == 1 || IsKeyMatch(msg, "backspace") {
		if IsKeyMatch(msg, "backspace") && len(m.TaskPickerInput) > 0 {
			// Remove last character
			m.TaskPickerInput = m.TaskPickerInput[:len(m.TaskPickerInput)-1]
		} else if !IsKeyMatch(msg, "backspace") {
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

// executeNextSelectedTask executes the task at the given index and then executes the next task
func (m Model) executeNextSelectedTask(index int) (Model, tea.Cmd) {
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
	return m, func() tea.Msg {
		task.ExecuteTask(selectedTask.Id, m.MessageBus)
		return TickMessage{}
	}
}

// RefreshTaskList refreshes the task list
func (m *Model) RefreshTaskList() tea.Cmd {
	m.Tasks = []task.Task{}
	m.TasksLoading = true
	m.AppendAppMsg("\nRefreshing task list\n")
	return func() tea.Msg {
		task.ListAllJson(m.MessageBus)
		return TickMessage{}
	}
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

	// Resize help viewport if needed
	if m.State == StateHelpOverlay {
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

	return func() tea.Msg {
		task.ExecuteTask(selectedTask.Id, m.MessageBus)
		return TickMessage{}
	}
}
