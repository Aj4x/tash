package task

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/Aj4x/tash/internal/msgbus"
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

type Type string

func (t Type) Topic() msgbus.Topic {
	return msgbus.Topic(t)
}

func (t Type) Message() Message {
	return Message{
		Type: t,
		ctx:  context.Background(),
	}
}

const (
	TypeTaskOutput      = Type("task.output")
	TypeTaskOutputErr   = Type("task.outputerr")
	TypeTaskError       = Type("task.error")
	TypeTaskJSON        = Type("task.json")
	TypeTaskCommand     = Type("task.command")
	TypeTaskDone        = Type("task.done")
	TypeTaskListAllDone = Type("list.done")
	TypeTaskListAllErr  = Type("list.error")
)

type Message struct {
	Type      Type
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (m Message) TopicMessage() msgbus.TopicMessage[Message] {
	return msgbus.TopicMessage[Message]{
		Topic:   m.Type.Topic(),
		Message: m,
	}
}

type ContextKey string

const (
	CtxKeyError       = ContextKey("error")
	CtxKeyOutput      = ContextKey("output")
	CtxKeyCommand     = ContextKey("command")
	CtxKeyTaskRunning = ContextKey("taskRunning")
)

func (m Message) Error() error {
	return m.ctx.Value(CtxKeyError).(error)
}

func (m Message) SetError(err error) Message {
	m.ctx = context.WithValue(m.ctx, CtxKeyError, err)
	return m
}

func (m Message) Output() string {
	return m.ctx.Value(CtxKeyOutput).(string)
}

func (m Message) SetOutput(output string) Message {
	m.ctx = context.WithValue(m.ctx, CtxKeyOutput, output)
	return m
}

func (m Message) Command() *exec.Cmd {
	return m.ctx.Value(CtxKeyCommand).(*exec.Cmd)
}

func (m Message) SetCommand(cmd *exec.Cmd) Message {
	m.ctx = context.WithValue(m.ctx, CtxKeyCommand, cmd)
	return m
}

func (m Message) TaskRunning() bool {
	val := m.ctx.Value(CtxKeyTaskRunning)
	if val == nil {
		return false
	}
	return val.(bool)
}

func (m Message) SetTaskRunning(isRunning bool) Message {
	m.ctx = context.WithValue(m.ctx, CtxKeyTaskRunning, isRunning)
	return m
}

func (m Message) Wait() {
	if m.Type != TypeTaskCommand {
		return
	}
	<-m.ctx.Done()
}

func (m Message) Cancel() {
	if m.Type != TypeTaskCommand {
		return
	}
	m.ctxCancel()
}

// ListAllJson executes the "task --list-all --json" command and sends the resulting JSON to the message bus.
func ListAllJson(bus msgbus.Publisher[Message]) {
	cmd := exec.Command("task", "--list-all", "--json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		bus.Publish(TypeTaskListAllErr.Message().SetError(err).TopicMessage())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		bus.Publish(TypeTaskListAllErr.Message().SetError(err).TopicMessage())
		return
	}
	if err := cmd.Start(); err != nil {
		bus.Publish(TypeTaskListAllErr.Message().SetError(err).TopicMessage())
		return
	}
	var taskOut string
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
			bus.Publish(TypeTaskOutputErr.Message().SetOutput(stdErrScanner.Text()).TopicMessage())
		}
		if started {
			wg.Done()
		}
	}()
	wg.Add(1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			bus.Publish(TypeTaskOutputErr.
				Message().
				SetOutput(fmt.Sprintf("error getting task list: %s", err)).
				TopicMessage(),
			)
			wg.Done()
		}
		wg.Done()
	}()
	wg.Wait()
	if len(taskOut) > 0 {
		bus.Publish(TypeTaskJSON.Message().SetOutput(taskOut).TopicMessage())
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

// ExecuteTask runs a task
func ExecuteTask(taskId string, bus msgbus.Publisher[Message]) {
	command := exec.Command("task", taskId)
	command.SysProcAttr = TaskProcessAttr()
	bus.Publish(TypeTaskCommand.Message().SetCommand(command).SetTaskRunning(true).TopicMessage())
	stdout, err := command.StdoutPipe()
	if err != nil {
		bus.Publish(TypeTaskCommand.Message().SetCommand(nil).SetTaskRunning(false).TopicMessage())
		bus.Publish(TypeTaskError.Message().SetError(err).TopicMessage())
		return
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		bus.Publish(TypeTaskCommand.Message().SetCommand(nil).SetTaskRunning(false).TopicMessage())
		bus.Publish(TypeTaskError.Message().SetError(err).TopicMessage())
		return
	}
	if err := command.Start(); err != nil {
		bus.Publish(TypeTaskCommand.Message().SetCommand(nil).SetTaskRunning(false).TopicMessage())
		bus.Publish(TypeTaskError.Message().SetError(err).TopicMessage())
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			bus.Publish(TypeTaskOutput.Message().SetOutput(scanner.Text()).TopicMessage())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			bus.Publish(TypeTaskOutputErr.Message().SetOutput(scanner.Text()).TopicMessage())
		}
	}()

	err = command.Wait()

	bus.Publish(TypeTaskCommand.Message().SetCommand(nil).SetTaskRunning(false).TopicMessage())

	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			err = fmt.Errorf("task failed with exit code %d: %w", exitError.ExitCode(), err)
			bus.Publish(TypeTaskError.Message().SetError(err).TopicMessage())
			return
		}
		err = fmt.Errorf("task failed: %w", err)
		bus.Publish(TypeTaskError.Message().SetError(err).TopicMessage())
		return
	}

	bus.Publish(TypeTaskDone.Message().TopicMessage())
}
