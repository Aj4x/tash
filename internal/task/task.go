package task

import (
	"bufio"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"os/exec"
	"strings"
)

// Task represents a task from the Taskfile
type Task struct {
	Id      string   `json:"id"`
	Desc    string   `json:"desc,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
}

// Message types for Bubble Tea
type ListAllErrMsg struct{ Err error }
type ListAllDoneMsg struct{}
type TaskMsg string
type TaskErrMsg struct{ Err error }
type TaskOutputMsg string
type TaskOutputErrMsg string
type TaskDoneMsg struct{}
type TaskCommandMsg struct {
	Command     *exec.Cmd
	TaskRunning bool
}

// ListAll executes the task --list-all command and sends the output to the target channel
func ListAll(target chan string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("task", "--list-all")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return ListAllErrMsg{Err: err}
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return ListAllErrMsg{Err: err}
		}
		if err := cmd.Start(); err != nil {
			return ListAllErrMsg{Err: err}
		}
		reader := io.MultiReader(stdout, stderr)
		buf := bufio.NewReader(reader)
		for {
			line, _, err := buf.ReadLine()
			if err == io.EOF {
				return ListAllDoneMsg{}
			}
			if err != nil {
				return ListAllErrMsg{Err: err}
			}
			target <- string(line)
		}
	}
}

// ParseTaskLine parses a task line from the task --list-all output
func ParseTaskLine(taskMsg string) (Task, bool) {
	line, ok := strings.CutPrefix(taskMsg, "* ")
	if !ok {
		return Task{}, false
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
	return Task{
		Id:      id,
		Desc:    strings.TrimSpace(desc),
		Aliases: aliases,
	}, true
}

// ExecuteTask runs a task and returns a command that will handle the output
func ExecuteTask(taskId string, target chan string, cmdChan chan TaskCommandMsg, errs chan string) tea.Cmd {
	return func() tea.Msg {
		command := exec.Command("task", taskId)
		command.SysProcAttr = TaskProcessAttr()
		cmdChan <- TaskCommandMsg{Command: command, TaskRunning: true}
		stdout, err := command.StdoutPipe()
		if err != nil {
			cmdChan <- TaskCommandMsg{Command: nil, TaskRunning: false}
			return TaskErrMsg{Err: err}
		}
		stderr, err := command.StderrPipe()
		if err != nil {
			cmdChan <- TaskCommandMsg{Command: nil, TaskRunning: false}
			return TaskErrMsg{Err: err}
		}
		if err := command.Start(); err != nil {
			cmdChan <- TaskCommandMsg{Command: nil, TaskRunning: false}
			return TaskErrMsg{Err: err}
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

		err = command.Wait()

		cmdChan <- TaskCommandMsg{Command: nil, TaskRunning: false}

		if err != nil {
			cmdChan <- TaskCommandMsg{Command: nil, TaskRunning: false}
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				return TaskErrMsg{Err: fmt.Errorf("task failed with exit code %d: %w", exitError.ExitCode(), err)}
			}
			return TaskErrMsg{Err: fmt.Errorf("task failed: %w", err)}
		}

		return TaskDoneMsg{}
	}
}
