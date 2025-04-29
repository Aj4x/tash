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

func (t Type) topic() msgbus.Topic {
	return msgbus.Topic(t)
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

type ContextKey string

const (
	CtxKeyError       = ContextKey("error")
	CtxKeyOutput      = ContextKey("output")
	CtxKeyCommand     = ContextKey("command")
	CtxKeyTaskRunning = ContextKey("taskRunning")
)

func (m *Message) Error() error {
	return m.ctx.Value(CtxKeyError).(error)
}

func (m *Message) Output() string {
	return m.ctx.Value(CtxKeyOutput).(string)
}

func (m *Message) Command() *exec.Cmd {
	return m.ctx.Value(CtxKeyCommand).(*exec.Cmd)
}

func (m *Message) TaskRunning() bool {
	return m.ctx.Value(CtxKeyTaskRunning).(bool)
}

func (m *Message) Wait() {
	if m.Type != TypeTaskCommand {
		return
	}
	<-m.ctx.Done()
}

func (m *Message) Cancel() {
	if m.Type != TypeTaskCommand {
		return
	}
	m.ctxCancel()
}

func NewCommandMessage(ctx context.Context, cmd *exec.Cmd) Message {
	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, CtxKeyCommand, cmd)
	ctx = context.WithValue(ctx, CtxKeyTaskRunning, cmd != nil)
	return Message{
		Type:      TypeTaskCommand,
		ctx:       ctx,
		ctxCancel: cancel,
	}
}

func NewOutputMessage(ctx context.Context, output string) Message {
	ctx = context.WithValue(ctx, CtxKeyOutput, output)
	return Message{
		Type: TypeTaskOutput,
		ctx:  ctx,
	}
}

func NewOutputErrMessage(ctx context.Context, output string) Message {
	ctx = context.WithValue(ctx, CtxKeyOutput, output)
	return Message{
		Type: TypeTaskOutputErr,
		ctx:  ctx,
	}
}

func NewErrorMessage(ctx context.Context, err error) Message {
	ctx = context.WithValue(ctx, CtxKeyError, err)
	return Message{
		Type: TypeTaskError,
		ctx:  ctx,
	}
}

func NewTaskJsonMessage(ctx context.Context, output string) Message {
	ctx = context.WithValue(ctx, CtxKeyOutput, output)
	return Message{
		Type: TypeTaskJSON,
		ctx:  ctx,
	}
}

const (
	TopicTaskOutput      = msgbus.Topic("task.output")
	TopicTaskOutputErr   = msgbus.Topic("task.outputerr")
	TopicTaskError       = msgbus.Topic("task.error")
	TopicTaskJSON        = msgbus.Topic("task.json")
	TopicTaskCommand     = msgbus.Topic("task.command")
	TopicTaskDone        = msgbus.Topic("task.done")
	TopicTaskListAllDone = msgbus.Topic("list.done")
	TopicTaskListAllErr  = msgbus.Topic("list.error")
)

// ListAllJson executes the "task --list-all --json" command and sends the resulting JSON to the message bus.
func ListAllJson(bus msgbus.Publisher[Message]) {
	publishListErr := func(err error) {
		bus.Publish(msgbus.TopicMessage[Message]{
			Topic:   TopicTaskListAllErr,
			Message: NewErrorMessage(context.Background(), err),
		})
	}
	publishListOutputErr := func(output string) {
		bus.Publish(msgbus.TopicMessage[Message]{
			Topic:   TopicTaskOutputErr,
			Message: NewOutputErrMessage(context.Background(), output),
		})
	}
	publishTaskJson := func(output string) {
		bus.Publish(msgbus.TopicMessage[Message]{
			Topic:   TopicTaskJSON,
			Message: NewTaskJsonMessage(context.Background(), output),
		})
	}
	cmd := exec.Command("task", "--list-all", "--json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		publishListErr(err)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		publishListErr(err)
		return
	}
	if err := cmd.Start(); err != nil {
		publishListErr(err)
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
			publishListOutputErr(stdErrScanner.Text())
		}
		if started {
			wg.Done()
		}
	}()
	wg.Add(1)
	go func() {
		err := cmd.Wait()
		if err != nil {
			publishListOutputErr(fmt.Sprintf("error getting task list: %s", err))
			wg.Done()
		}
		wg.Done()
	}()
	wg.Wait()
	if len(taskOut) > 0 {
		publishTaskJson(taskOut)
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
	publishCommandStopped := func() {
		bus.Publish(msgbus.TopicMessage[Message]{
			Topic:   TopicTaskCommand,
			Message: NewCommandMessage(context.Background(), nil),
		})
	}
	publishTaskErr := func(err error) {
		bus.Publish(msgbus.TopicMessage[Message]{
			Topic:   TopicTaskError,
			Message: NewErrorMessage(context.Background(), err),
		})
	}
	publishTaskOutput := func(output string) {
		bus.Publish(msgbus.TopicMessage[Message]{
			Topic:   TopicTaskOutput,
			Message: NewOutputMessage(context.Background(), output),
		})
	}
	publishTaskOutputErr := func(output string) {
		bus.Publish(msgbus.TopicMessage[Message]{
			Topic:   TopicTaskOutputErr,
			Message: NewOutputErrMessage(context.Background(), output),
		})
	}
	command := exec.Command("task", taskId)
	command.SysProcAttr = TaskProcessAttr()
	bus.Publish(msgbus.TopicMessage[Message]{
		Topic:   TopicTaskCommand,
		Message: NewCommandMessage(context.Background(), command),
	})
	stdout, err := command.StdoutPipe()
	if err != nil {
		publishCommandStopped()
		publishTaskErr(err)
		return
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		publishCommandStopped()
		publishTaskErr(err)
		return
	}
	if err := command.Start(); err != nil {
		publishCommandStopped()
		publishTaskErr(err)
		return
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			publishTaskOutput(scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			publishTaskOutputErr(scanner.Text())
		}
	}()

	err = command.Wait()

	publishCommandStopped()

	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			publishTaskErr(fmt.Errorf("task failed with exit code %d: %w", exitError.ExitCode(), err))
			return
		}
		publishTaskErr(fmt.Errorf("task failed: %w", err))
		return
	}

	bus.Publish(msgbus.TopicMessage[Message]{
		Topic:   TopicTaskDone,
		Message: Message{Type: TypeTaskDone},
	})
}
