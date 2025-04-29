package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Aj4x/tash/internal/task"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleBusMessage(message task.Message) (Model, tea.Cmd) {
	switch message.Type {
	case task.TypeTaskOutput:
		return m.handleTaskOutputMsg(message)
	case task.TypeTaskOutputErr:
		return m.handleTaskOutputErr(message)
	case task.TypeTaskError:
		return m.handleTaskErrorMsg(message)
	case task.TypeTaskJSON:
		return m.handleTaskJsonMsg(message)
	case task.TypeTaskCommand:
		return m.handleTaskCommandMsg(message)
	case task.TypeTaskDone:
		return m.handleTaskDoneMsg(message)
	case task.TypeTaskListAllDone:
		return m.handleListAllDoneMsg(message)
	case task.TypeTaskListAllErr:
		return m.handleListAllErrMsg(message)
	default:
		return m, nil
	}
}

func (m Model) handleTaskOutputMsg(msg task.Message) (Model, tea.Cmd) {
	m.AppendCommandOutput(msg.Output())
	return m, nil
}

func (m Model) handleTaskOutputErr(msg task.Message) (Model, tea.Cmd) {
	m.AppendErrorMsg(msg.Output())
	return m, nil
}

func (m Model) handleTaskErrorMsg(msg task.Message) (Model, tea.Cmd) {
	m.AppendErrorMsg(msg.Error().Error())
	if m.ExecutingBatch {
		m.AppendErrorMsg("Batch execution aborted")
		m.ExecutingBatch = false
		m.CurrentBatchTaskIndex = -1
	}
	return m, nil
}

// handleTaskCommandMsg processes task command messages
func (m Model) handleTaskCommandMsg(msg task.Message) (Model, tea.Cmd) {
	m.TaskRunning = msg.TaskRunning()
	m.Command = msg.Command()
	return m, nil
}

// handleTaskJsonMsg processes task JSON messages
func (m Model) handleTaskJsonMsg(msg task.Message) (Model, tea.Cmd) {
	msgContent := msg.Output()
	tasks, err := parseTasksJson(msgContent)
	if err != nil {
		m.AppendErrorMsg("Error parsing task list: " + err.Error())
		return m, nil
	}
	var parsedJson bytes.Buffer
	err = json.Indent(&parsedJson, []byte(msgContent), "", "\t")
	if err != nil {
		m.AppendErrorMsg("Error parsing json output for printing: " + err.Error())
		m.AppendErrorMsg(msgContent)
	} else {
		m.AppendCommandOutput(string(parsedJson.Bytes()))
	}
	m.AppendAppMsg(fmt.Sprintf("Task list:\n%s\n", parsedJson.String()))
	m.Tasks = tasks
	m.AppendAppMsg(fmt.Sprintf("Tasks added: %d\n", len(m.Tasks)))
	m.UpdateTaskTable()
	m.TasksLoading = false
	return m, nil
}

func (m Model) handleTaskDoneMsg(msg task.Message) (Model, tea.Cmd) {
	m.TasksLoading = false
	m.AppendAppMsg("Task executed successfully!\n")
	if m.ExecutingBatch {
		return m.executeNextSelectedTask(m.CurrentBatchTaskIndex)
	}
	return m, nil
}

func (m Model) handleListAllDoneMsg(msg task.Message) (Model, tea.Cmd) {
	m.TasksLoading = false
	m.AppendAppMsg("Task list refreshed successfully!\n")
	m.UpdateTaskTable()
	return m, nil
}

func (m Model) handleListAllErrMsg(msg task.Message) (Model, tea.Cmd) {
	m.TasksLoading = false
	m.AppendErrorMsg("Error: " + msg.Error().Error())
	return m, nil
}
