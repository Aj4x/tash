package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
)

var (
	tableStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	viewportStyle = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	focusedStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("69"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Message Styles
	appMsgStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // Green for app messages
	errorMsgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // Red for error messages
	outputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))             // Default color for regular output
)

func main() {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("tash error: " + err.Error())
		os.Exit(1)
	}
}

type Task struct {
	Id      string
	Desc    string
	Aliases []string
}

type Control int

const (
	ControlTable Control = iota
	ControlViewport

	// put any new controls above this line, so our ControlMax, used for tabbing
	// though the controls, stays the maximum Control value
	ControlMax
)

func (c Control) Tab() Control {
	tabbedControl := Control(int(c) + 1)
	if tabbedControl >= ControlMax {
		return Control(0)
	}
	return tabbedControl
}

type Model struct {
	Tasks              []Task
	tasksLoading       bool
	result             *string
	taskChan           chan string
	outChan            chan string
	errChan            chan string
	cmdChan            chan TaskCommandMsg
	viewport           viewport.Model
	table              table.Model
	focused            Control
	width              int
	height             int
	initialised        bool
	selectedTask       *Task
	showDetailsOverlay bool
	command            *exec.Cmd
	taskRunning        bool
}

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
		Tasks:              []Task{},
		result:             new(string),
		taskChan:           make(chan string),
		outChan:            make(chan string),
		errChan:            make(chan string),
		cmdChan:            make(chan TaskCommandMsg),
		viewport:           viewport.New(0, 0),
		table:              t,
		focused:            ControlTable,
		initialised:        false,
		selectedTask:       nil,
		showDetailsOverlay: false,
	}
}

func (m Model) Init() tea.Cmd {
	return m.RefreshTaskList()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.initialised {
			m.initialised = true
		}
		m.width = msg.Width
		m.height = msg.Height

		tableWidth := int(float64(m.width) * 0.4)
		viewportWidth := m.width - tableWidth - 2

		m.table.SetWidth(tableWidth)
		m.table.SetHeight(m.height - 4)

		m.viewport.Width = viewportWidth
		m.viewport.Height = m.height - 4
		m.viewport.HighPerformanceRendering = false

		return m, nil
	case tea.KeyMsg:
		if m.showDetailsOverlay {
			if msg.String() == "esc" || msg.String() == "i" {
				m.showDetailsOverlay = false
				return m, nil
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+l":
			if m.tasksLoading {
				return m, nil
			}
			m.result = new(string)
			m.viewport.SetContent(*m.result)
			m.viewport.GotoTop()
			return m, nil
		case "ctrl+r":
			if m.tasksLoading {
				return m, nil
			}
			return m, m.RefreshTaskList()
		case "i":
			if m.focused == ControlTable && len(m.Tasks) > 0 && m.table.SelectedRow() != nil {
				selectedIndex := m.table.Cursor()
				m.selectedTask = &m.Tasks[selectedIndex]
				m.showDetailsOverlay = true
			}
			return m, nil
		case "tab":
			m.focused = m.focused.Tab()
			if m.focused == ControlTable {
				m.table.Focus()
			} else {
				m.table.Blur()
			}
			return m, nil
		case "up", "down", "j", "k":
			switch m.focused {
			case ControlTable:
				var cmd tea.Cmd
				m.table, cmd = m.table.Update(msg)
				cmds = append(cmds, cmd)
				break
			case ControlViewport:
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			default:
				return m, nil
			}
			return m, tea.Batch(cmds...)
		case "enter", "e":
			if m.tasksLoading {
				return m, nil
			}
			if m.focused == ControlTable && len(m.Tasks) > 0 && m.table.SelectedRow() != nil {
				selectedIndex := m.table.Cursor()
				selectedTask := m.Tasks[selectedIndex]
				m.appendAppMsg(fmt.Sprintf("Executing task: %s\n\n", selectedTask.Id))
				m.tasksLoading = true
				return m, tea.Batch(
					m.ExecuteTask(selectedTask.Id, m.outChan, m.cmdChan, m.errChan),
					m.waitForTaskCommandMsg(),
					m.waitForTaskOutputMsg(),
					m.waitForTaskErrorMsg(),
				)
			}
			return m, nil
		case "ctrl+x":
			if m.taskRunning {
				if err := StopTaskProcess(m.command.Process); err != nil {
					m.appendErrorMsg("Error cancelling task: " + err.Error())
					return m, nil
				}
				m.taskRunning = false
				m.command = nil
				m.appendAppMsg("Task cancelled\n")
			}
		}
	case TaskCommandMsg:
		m.taskRunning = msg.taskRunning
		m.command = msg.command
		cmds = append(cmds, m.waitForTaskCommandMsg())
		if msg.taskRunning {
			m.appendAppMsg("TaskCommandMsg: running\n")
		}
		if msg.taskRunning {
			m.appendAppMsg("TaskCommandMsg: stopped\n")
		}
	case TaskMsg:
		m.appendCommandOutput(string(msg))
		cmds = append(cmds, m.waitForTaskMsg())
		m.appendTask(string(msg))
		slices.SortFunc(m.Tasks, func(a, b Task) int {
			return strings.Compare(a.Id, b.Id)
		})
	case TaskOutputMsg:
		m.appendCommandOutput(string(msg))
		cmds = append(cmds, m.waitForTaskOutputMsg())
	case TaskOutputErrMsg:
		m.appendErrorMsg(string(msg))
		cmds = append(cmds, m.waitForTaskErrorMsg())
	case TaskErrMsg:
		m.appendErrorMsg(msg.err.Error())
		cmds = append(cmds, m.waitForTaskMsg())
	case ListAllErrMsg:
		m.tasksLoading = false
		m.appendErrorMsg("Error: " + msg.err.Error())
	case TaskDoneMsg:
		m.tasksLoading = false
		m.appendAppMsg("Task executed successfully!\n")
	case ListAllDoneMsg:
		m.tasksLoading = false
		m.appendAppMsg("Task list refreshed successfully!\n")
		m.updateTaskTable()
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.initialised {
		return "Initialising..."
	}

	tableRendered := m.table.View()
	viewportRendered := m.viewport.View()

	if m.focused == ControlTable {
		tableRendered = focusedStyle.Render(tableRendered)
		viewportRendered = viewportStyle.Render(viewportRendered)
	} else if m.focused == ControlViewport {
		tableRendered = tableStyle.Render(tableRendered)
		viewportRendered = focusedStyle.Render(viewportRendered)
	}

	// Build the layout
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, tableRendered, viewportRendered)

	// Add help text at the bottom
	helpText := helpStyle.Render(m.helpView())

	fullView := lipgloss.JoinVertical(lipgloss.Left, mainView, helpText)

	// If overlay is visible, render it on top
	if m.showDetailsOverlay {
		return m.renderTaskDetailOverlay()
	}

	return fullView
}

func (m Model) helpView() string {
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
	if m.taskRunning {
		help = append(help, "ctrl+x: cancel task")
	}

	return helpStyle.Render(strings.Join(help, " • "))
}

func (m *Model) RefreshTaskList() tea.Cmd {
	m.Tasks = []Task{}
	m.tasksLoading = true
	m.appendAppMsg("\nRefreshing task list\n")
	return tea.Batch(ListAll(m.taskChan), m.waitForTaskMsg())
}

type ListAllErrMsg struct{ err error }
type ListAllDoneMsg struct{}

func ListAll(target chan string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("task", "--list-all")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return ListAllErrMsg{err: err}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return ListAllErrMsg{err: err}
		}
		if err := cmd.Start(); err != nil {
			return ListAllErrMsg{err: err}
		}
		reader := io.MultiReader(stdout, stderr)
		buf := bufio.NewReader(reader)
		for {
			line, _, err := buf.ReadLine()
			if err == io.EOF {
				return ListAllDoneMsg{}
			}
			if err != nil {
				return ListAllErrMsg{err: err}
			}
			target <- string(line)
		}
	}
}

type TaskMsg string
type TaskErrMsg struct{ err error }
type TaskOutputMsg string
type TaskOutputErrMsg string
type TaskDoneMsg struct{}
type TaskCommandMsg struct {
	command     *exec.Cmd
	taskRunning bool
}

func (m Model) waitForTaskMsg() tea.Cmd {
	return func() tea.Msg {
		return TaskMsg(<-m.taskChan)
	}
}

func (m Model) waitForTaskOutputMsg() tea.Cmd {
	return func() tea.Msg {
		return TaskOutputMsg(<-m.outChan)
	}
}

func (m Model) waitForTaskErrorMsg() tea.Cmd {
	return func() tea.Msg {
		return TaskOutputErrMsg(<-m.errChan)
	}
}

func (m Model) waitForTaskCommandMsg() tea.Cmd {
	return func() tea.Msg {
		msg := <-m.cmdChan
		return msg
	}
}

func (m *Model) appendTask(taskMsg string) {
	line, ok := strings.CutPrefix(taskMsg, "* ")
	if !ok {
		return
	}
	t := Task{}
	id, line, ok := strings.Cut(line, ": ")
	if ok {
		t.Id = id
	}
	var aliases []string
	desc, aliasStr, ok := strings.Cut(line, "(aliases:")
	if ok {
		aliasStr, _ = strings.CutSuffix(aliasStr, ")")
		aliases = strings.Split(aliasStr, ",")
	}
	m.Tasks = append(m.Tasks, Task{
		Id:      id,
		Desc:    strings.TrimSpace(desc),
		Aliases: aliases,
	})
}

func (m *Model) updateTaskTable() {
	var rows []table.Row
	for _, task := range m.Tasks {
		rows = append(rows, table.Row{
			task.Id,
			task.Desc,
		})
	}
	m.table.SetRows(rows)
}

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

func (m *Model) appendToViewport(msg string) {
	lines := TextWrap(msg, m.viewport.Width-2)
	for _, line := range lines {
		*m.result += "\n" + line
	}
	//*m.result += "\n" + msg
	m.viewport.SetContent(*m.result)
	m.viewport.GotoBottom()
}

func (m *Model) appendAppMsg(msg string) {
	m.appendToViewport(appMsgStyle.Render(msg))
}

func (m *Model) appendErrorMsg(msg string) {
	m.appendToViewport(errorMsgStyle.Render(msg))
}

func (m *Model) appendCommandOutput(msg string) {
	m.appendToViewport(outputStyle.Render(msg))
}

func (m Model) renderTaskDetailOverlay() string {
	if !m.showDetailsOverlay || m.selectedTask == nil {
		return ""
	}

	// Calculate overlay dimensions
	width := int(float64(m.width) * 0.7)
	height := int(float64(m.height) * 0.7)

	// Create styles for overlay components
	overlayStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Width(width).
		Height(height)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("241"))

	// Format aliases as a comma-separated list
	aliases := strings.Join(m.selectedTask.Aliases, ", ")

	// Build the content
	content := titleStyle.Render("Task Details") + "\n\n"
	content += labelStyle.Render("ID: ") + m.selectedTask.Id + "\n\n"
	content += labelStyle.Render("Description: ") + m.selectedTask.Desc + "\n\n"
	content += labelStyle.Render("Aliases: ") + aliases + "\n"

	// Wrap the content in the overlay style
	overlay := overlayStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		overlay,
	)
}

// ExecuteTask runs a task and returns a command that will handle the output
func (m *Model) ExecuteTask(taskId string, target chan string, cmdChan chan TaskCommandMsg, errs chan string) tea.Cmd {
	return func() tea.Msg {
		m.taskRunning = true
		m.command = exec.Command("task", taskId)
		m.command.SysProcAttr = TaskProcessAttr()
		cmdChan <- TaskCommandMsg{command: m.command, taskRunning: true}
		stdout, err := m.command.StdoutPipe()
		if err != nil {
			m.taskRunning = false
			m.command = nil
			cmdChan <- TaskCommandMsg{command: nil, taskRunning: false}
			return TaskErrMsg{err: err}
		}
		stderr, err := m.command.StderrPipe()
		if err != nil {
			m.taskRunning = false
			m.command = nil
			cmdChan <- TaskCommandMsg{command: nil, taskRunning: false}
			return TaskErrMsg{err: err}
		}
		if err := m.command.Start(); err != nil {
			m.taskRunning = false
			m.command = nil
			cmdChan <- TaskCommandMsg{command: nil, taskRunning: false}
			return TaskErrMsg{err: err}
		}

		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				target <- line
			}
		}()

		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				errs <- line
			}
		}()

		err = m.command.Wait()

		m.taskRunning = false
		m.command = nil
		cmdChan <- TaskCommandMsg{command: nil, taskRunning: false}

		if err != nil {
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				return TaskErrMsg{err: fmt.Errorf("task failed with exit code %d: %w", exitError.ExitCode(), err)}
			}
			return TaskErrMsg{err: fmt.Errorf("task failed: %w", err)}
		}

		return TaskDoneMsg{}

		//reader := io.MultiReader(stdout, stderr)
		//buf := bufio.NewReader(reader)
		//
		//for {
		//	line, _, err := buf.ReadLine()
		//	if err == io.EOF {
		//		return TaskDoneMsg{}
		//	}
		//	if err != nil {
		//		return TaskErrMsg{err: err}
		//	}
		//	target <- string(line)
		//}
	}
}
