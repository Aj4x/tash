package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"os/exec"
	"slices"
	"strings"

	"github.com/Aj4x/tash/internal/task"
)

var (
	TableStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	ViewportStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	FocusedStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("69"))
	HelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Message Styles
	AppMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // Green for app messages
	ErrorMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // Red for error messages
	OutputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))             // Default color for regular output
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
func RenderHelpView(taskRunning bool) string {
	help := []string{
		"q/esc: quit",
		"tab: switch focus",
		"↑/↓/j/k: navigate",
		"enter/e: execute task",
		"i: task details",
		"ctrl+r: refresh tasks",
		"ctrl+l: clear output",
	}

	// Add the Ctrl+x help text only when a task is running
	if taskRunning {
		help = append(help, "ctrl+x: cancel task")
	}

	return HelpStyle.Render(strings.Join(help, " • "))
}

// RenderTaskDetailOverlay renders an overlay with detailed task information
func RenderTaskDetailOverlay(width, height int, selectedTask *task.Task) string {
	if selectedTask == nil {
		return ""
	}

	// Calculate overlay dimensions
	overlayWidth := int(float64(width) * 0.7)
	overlayHeight := int(float64(height) * 0.7)

	// Create styles for overlay components
	overlayStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(overlayWidth).
		Height(overlayHeight)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("241"))

	// Format aliases as a comma-separated list
	aliases := strings.Join(selectedTask.Aliases, ", ")

	// Build the content
	content := titleStyle.Render("Task Details") + "\n\n"
	content += labelStyle.Render("ID: ") + selectedTask.Id + "\n\n"
	content += labelStyle.Render("Description: ") + selectedTask.Desc + "\n\n"
	content += labelStyle.Render("Aliases: ") + aliases + "\n"

	// Wrap the content in the overlay style
	overlay := overlayStyle.Render(content)

	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
}

// Model represents the UI model for the application
type Model struct {
	Tasks              []task.Task
	TasksLoading       bool
	Result             *string
	TaskChan           chan string
	OutChan            chan string
	ErrChan            chan string
	CmdChan            chan task.TaskCommandMsg
	Viewport           viewport.Model
	Table              table.Model
	Focused            Control
	Width              int
	Height             int
	Initialised        bool
	SelectedTask       *task.Task
	ShowDetailsOverlay bool
	Command            *exec.Cmd
	TaskRunning        bool
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
		Header:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")),
		Selected: lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true),
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

	// Add help text at the bottom
	helpText := RenderHelpView(m.TaskRunning)

	fullView := lipgloss.JoinVertical(lipgloss.Left, mainView, helpText)

	// If overlay is visible, render it on top
	if m.ShowDetailsOverlay {
		return RenderTaskDetailOverlay(m.Width, m.Height, m.SelectedTask)
	}

	return fullView
}

// AppendToViewport adds text to the viewport
func (m *Model) AppendToViewport(msg string) {
	lines := TextWrap(msg, m.Viewport.Width-2)
	for _, line := range lines {
		*m.Result += "\n" + line
	}
	m.Viewport.SetContent(*m.Result)
	m.Viewport.GotoBottom()
}

// AppendAppMsg adds an application message to the viewport
func (m *Model) AppendAppMsg(msg string) {
	m.AppendToViewport(AppMsgStyle.Render(msg))
}

// AppendErrorMsg adds an error message to the viewport
func (m *Model) AppendErrorMsg(msg string) {
	m.AppendToViewport(ErrorMsgStyle.Render(msg))
}

// AppendCommandOutput adds command output to the viewport
func (m *Model) AppendCommandOutput(msg string) {
	m.AppendToViewport(OutputStyle.Render(msg))
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
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.Initialised {
			m.Initialised = true
		}
		m.HandleWindowResize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if m.ShowDetailsOverlay {
			if msg.String() == "esc" || msg.String() == "i" {
				m.ShowDetailsOverlay = false
				return m, nil
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+l":
			if m.TasksLoading {
				return m, nil
			}
			m.Result = new(string)
			m.Viewport.SetContent(*m.Result)
			m.Viewport.GotoTop()
			return m, nil
		case "ctrl+r":
			if m.TasksLoading {
				return m, nil
			}
			return m, m.RefreshTaskList()
		case "i":
			if m.Focused == ControlTable && len(m.Tasks) > 0 && m.Table.SelectedRow() != nil {
				selectedIndex := m.Table.Cursor()
				m.SelectedTask = &m.Tasks[selectedIndex]
				m.ShowDetailsOverlay = true
			}
			return m, nil
		case "tab":
			m.Focused = m.Focused.Tab()
			if m.Focused == ControlTable {
				m.Table.Focus()
			} else {
				m.Table.Blur()
			}
			return m, nil
		case "up", "down", "j", "k":
			switch m.Focused {
			case ControlTable:
				var cmd tea.Cmd
				m.Table, cmd = m.Table.Update(msg)
				cmds = append(cmds, cmd)
				break
			case ControlViewport:
				var cmd tea.Cmd
				m.Viewport, cmd = m.Viewport.Update(msg)
				cmds = append(cmds, cmd)
			default:
				return m, nil
			}
			return m, tea.Batch(cmds...)
		case "enter", "e":
			if m.TasksLoading {
				return m, nil
			}
			if m.Focused == ControlTable && len(m.Tasks) > 0 && m.Table.SelectedRow() != nil {
				return m, m.ExecuteSelectedTask()
			}
			return m, nil
		case "ctrl+x":
			if m.TaskRunning {
				if err := task.StopTaskProcess(m.Command.Process); err != nil {
					m.AppendErrorMsg("Error cancelling task: " + err.Error())
					return m, nil
				}
				m.TaskRunning = false
				m.Command = nil
				m.AppendAppMsg("Task cancelled\n")
			}
		}
	case task.TaskCommandMsg:
		m.TaskRunning = msg.TaskRunning
		m.Command = msg.Command
		cmds = append(cmds, m.WaitForTaskCommandMsg())
		if msg.TaskRunning {
			m.AppendAppMsg("TaskCommandMsg: running\n")
		}
		if !msg.TaskRunning {
			m.AppendAppMsg("TaskCommandMsg: stopped\n")
		}
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
		cmds = append(cmds, m.WaitForTaskMsg())
	case task.ListAllErrMsg:
		m.TasksLoading = false
		m.AppendErrorMsg("Error: " + msg.Err.Error())
	case task.TaskDoneMsg:
		m.TasksLoading = false
		m.AppendAppMsg("Task executed successfully!\n")
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
	viewportWidth := m.Width - tableWidth - 2

	m.Table.SetWidth(tableWidth)
	m.Table.SetHeight(m.Height - 4)

	m.Viewport.Width = viewportWidth
	m.Viewport.Height = m.Height - 4
	m.Viewport.HighPerformanceRendering = false
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
