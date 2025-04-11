package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const viewportHeight = 12

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

	ControlMax = Control(iota)
)

func (c Control) Tab() Control {
	tabbedControl := Control(int(c) + 1)
	if tabbedControl >= ControlMax {
		return Control(0)
	}
	return tabbedControl
}

type Model struct {
	Tasks       []Task
	result      *string
	outChan     chan string
	viewport    viewport.Model
	table       table.Model
	focused     Control
	width       int
	height      int
	initialised bool
}

func NewModel() Model {
	columns := []table.Column{
		{Title: "Id", Width: 15},
		{Title: "Description", Width: 30},
		{Title: "Aliases", Width: 15},
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
		Tasks:       []Task{},
		result:      new(string),
		outChan:     make(chan string),
		viewport:    viewport.New(0, viewportHeight),
		table:       t,
		focused:     ControlTable,
		initialised: false,
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

		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+l":
			return m, m.RefreshTaskList()
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
		}
	case TaskMsg:
		m.appendCommandOutput(string(msg))
		cmds = append(cmds, m.waitForTaskMsg())
		m.appendTask(string(msg))
		slices.SortFunc(m.Tasks, func(a, b Task) int {
			return strings.Compare(a.Id, b.Id)
		})
	case ListAllErrMsg:
		m.appendErrorMsg("Error: " + msg.err.Error())
	case ListAllDoneMsg:
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

	if m.focused == 0 {
		tableRendered = focusedStyle.Render(tableRendered)
		viewportRendered = viewportStyle.Render(viewportRendered)
	} else {
		tableRendered = tableStyle.Render(tableRendered)
		viewportRendered = focusedStyle.Render(viewportRendered)
	}

	helpText := helpStyle.Render("↑/↓: Navigate • Tab: Switch focus • Ctrl+L: Refresh • q: Quit")

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableRendered,
		viewportRendered,
	) + "\n" + helpText
}

func (m *Model) RefreshTaskList() tea.Cmd {
	m.Tasks = []Task{}
	m.appendAppMsg("\nRefreshing task list\n")
	return tea.Batch(ListAll(m.outChan), m.waitForTaskMsg())
}

type ListAllErrMsg struct{ err error }
type ListAllDoneMsg struct{}

func ListAll(target chan string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("task", "--list-all")
		out, err := cmd.StdoutPipe()
		if err != nil {
			return ListAllErrMsg{err: err}
		}
		if err := cmd.Start(); err != nil {
			return ListAllErrMsg{err: err}
		}
		buf := bufio.NewReader(out)
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

func (m Model) waitForTaskMsg() tea.Cmd {
	return func() tea.Msg {
		return TaskMsg(<-m.outChan)
	}
}

func (m *Model) appendTask(taskMsg string) {
	line, ok := strings.CutPrefix(taskMsg, "* ")
	if !ok {
		return
	}
	t := Task{}
	id, line, ok := strings.Cut(line, ":")
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
		aliases := strings.Join(task.Aliases, ", ")
		rows = append(rows, table.Row{
			task.Id,
			task.Desc,
			aliases,
		})
	}
	m.table.SetRows(rows)
}

func (m *Model) appendToViewport(msg string) {
	*m.result += "\n" + msg
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
