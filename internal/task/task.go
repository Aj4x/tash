package task

import (
	"bufio"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os/exec"
	"strings"
	"sync"
)

// Task represents a task from the Taskfile
type Task struct {
	Id      string   `json:"name"`
	Desc    string   `json:"desc,omitempty"`
	Summary string   `json:"summary,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
}

// Message types for Bubble Tea
type ListAllErrMsg struct{ Err error }
type ListAllDoneMsg struct{}

type TaskJsonMsg string

func (t TaskJsonMsg) WaitForMessage(mo MessageObserver) tea.Cmd {
	return func() tea.Msg {
		return TaskJsonMsg(<-mo.TaskJsonChannel())
	}
}

type TaskErrMsg struct{ Err error }
type TaskOutputMsg string

func (t TaskOutputMsg) WaitForMessage(mo MessageObserver) tea.Cmd {
	return func() tea.Msg {
		return TaskOutputMsg(<-mo.OutputChannel())
	}
}

type TaskOutputErrMsg string

func (t TaskOutputErrMsg) WaitForMessage(mo MessageObserver) tea.Cmd {
	return func() tea.Msg {
		return TaskOutputErrMsg(<-mo.ErrorChannel())
	}
}

type TaskDoneMsg struct{}
type TaskCommandMsg struct {
	Command     *exec.Cmd
	TaskRunning bool
}

func (t TaskCommandMsg) WaitForMessage(mo MessageObserver) tea.Cmd {
	return func() tea.Msg {
		return <-mo.CommandChannel()
	}
}

type MessageListener interface {
	WaitForMessage(mo MessageObserver) tea.Cmd
}

type MessageObserver interface {
	TaskJsonChannel() chan string
	OutputChannel() chan string
	ErrorChannel() chan string
	CommandChannel() chan TaskCommandMsg
}

// ListAllJson executes the "task --list-all --json" command and sends the resulting JSON output to a target channel.
// Returns various `tea.Msg` types based on command execution success or failure.
func ListAllJson(target, errChan chan string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("task", "--list-all", "--json")
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
		var taskOut, errOut string
		stdoutScanner := bufio.NewScanner(stdout)
		stdErrScanner := bufio.NewScanner(stderr)
		wg := sync.WaitGroup{}
		go func() {
			started := false
			for stdoutScanner.Scan() {
				if !started {
					wg.Add(1)
					started = true
				}
				taskOut += stdoutScanner.Text()
			}
			if started {
				wg.Done()
			}
		}()
		go func() {
			started := false
			for stdErrScanner.Scan() {
				if !started {
					wg.Add(1)
					started = true
				}
				errChan <- stdErrScanner.Text()
			}
			if started {
				wg.Done()
			}
		}()
		wg.Add(1)
		go func() {
			err := cmd.Wait()
			if err != nil {
				errChan <- fmt.Sprintf("error getting task list: %s", err)
				wg.Done()
			}
			wg.Done()
		}()
		wg.Wait()
		if len(taskOut) > 0 {
			target <- taskOut
		}
		if len(errOut) > 0 {
			return TaskOutputErrMsg(errOut)
		}
		return ListAllDoneMsg{}
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
