package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const viewportHeight = 12

func main() {
	p := tea.NewProgram(NewModel())
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

type Model struct {
	Tasks    []Task
	result   *string
	outChan  chan string
	viewport viewport.Model
}

func NewModel() Model {
	return Model{
		Tasks:    []Task{},
		result:   new(string),
		outChan:  make(chan string),
		viewport: viewport.New(0, viewportHeight),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(ListAll(m.outChan), m.waitForTaskMsg(m.outChan))
	
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	case TaskMsg:
		m.appendOutput(string(msg))
		cmds = append(cmds, m.waitForTaskMsg(m.outChan))
		m.appendTask(string(msg))
	case ListAllErrMsg:
		m.appendOutput("Error: " + msg.err.Error())
	case ListAllDoneMsg:
		m.appendOutput("Done. Press `q` to exit.")
		m.appendOutput(fmt.Sprintf("Tasks: %v", m.Tasks))
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.viewport.View()
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

func (m Model) waitForTaskMsg(outChan chan string) tea.Cmd {
	return func() tea.Msg {
		return TaskMsg(<-outChan)
	}
}

func (m *Model) appendOutput(s string) {
	*m.result += "\n" + s
	m.viewport.SetContent(*m.result)
	m.viewport.GotoBottom()
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
