package ui

import (
	"fmt"

	"github.com/Aj4x/tash/internal/task"
	tea "github.com/charmbracelet/bubbletea"
)

// handleNormalKey handles key presses when in the normal state
func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Quit
	if IsKeyMatch(msg, "q") {
		return m, tea.Quit
	}

	// Clear output
	if IsKeyMatch(msg, "ctrl+l") {
		if m.TasksLoading {
			return m, nil
		}
		m.Result = new(string)
		m.Viewport.SetContent(*m.Result)
		m.Viewport.GotoTop()
		return m, nil
	}

	// Refresh tasks
	if IsKeyMatch(msg, "ctrl+r") {
		if m.TasksLoading {
			return m, nil
		}
		return m, m.RefreshTaskList()
	}

	// Task details
	if IsKeyMatch(msg, "i") {
		if m.Focused == ControlTable && len(m.Tasks) > 0 && m.Table.SelectedRow() != nil {
			selectedIndex := m.Table.Cursor()
			m.SelectedTask = &m.Tasks[selectedIndex]
			m.State = StateDetailsOverlay
		}
		return m, nil
	}

	// Switch focus
	if IsKeyMatch(msg, "tab") {
		m.Focused = m.Focused.Tab()
		if m.Focused == ControlTable {
			m.Table.Focus()
		} else {
			m.Table.Blur()
		}
		return m, nil
	}

	// Navigation
	if IsKeyMatch(msg, "↑/↓/j/k") {
		var cmds []tea.Cmd
		var cmd tea.Cmd

		switch m.Focused {
		case ControlTable:
			m.Table, cmd = m.Table.Update(msg)
			cmds = append(cmds, cmd)
		case ControlViewport:
			m.Viewport, cmd = m.Viewport.Update(msg)
			cmds = append(cmds, cmd)
		default:
			return m, nil
		}

		return m, tea.Batch(cmds...)
	}

	// Execute task
	if IsKeyMatch(msg, "enter/e") {
		if m.TasksLoading {
			return m, nil
		}

		if m.Focused == ControlTable && len(m.Tasks) > 0 && m.Table.SelectedRow() != nil {
			return m, m.ExecuteSelectedTask()
		}

		return m, nil
	}

	// Cancel task
	if IsKeyMatch(msg, "ctrl+x") {
		if m.TaskRunning {
			if err := task.StopTaskProcess(m.Command.Process); err != nil {
				m.AppendErrorMsg("Error cancelling task: " + err.Error())
				return m, nil
			}
			m.TaskRunning = false
			m.Command = nil
			m.AppendAppMsg("Task cancelled\n")
		}
		return m, nil
	}

	// Open task picker
	if IsKeyMatch(msg, "/") {
		if m.TasksLoading {
			return m, nil
		}

		m.State = StateTaskPicker
		m.TaskPickerInput = ""
		m.TaskPickerMatches = m.Tasks // Initialize with all tasks
		m.TaskPickerSelected = 0

		return m, nil
	}

	// Execute selected tasks
	if IsKeyMatch(msg, "ctrl+e") {
		if m.TasksLoading || len(m.SelectedTasks) == 0 || m.ExecutingBatch {
			return m, nil
		}

		m.ExecutingBatch = true
		m.CurrentBatchTaskIndex = 0

		m.AppendAppMsg(fmt.Sprintf("Executing %d selected tasks\n", len(m.SelectedTasks)))

		// Execute the first task
		return m.executeNextSelectedTask(m.CurrentBatchTaskIndex)
	}

	// Clear selected tasks
	if IsKeyMatch(msg, "ctrl+d") {
		if len(m.SelectedTasks) > 0 {
			m.SelectedTasks = []task.Task{}
			m.AppendAppMsg("Selected tasks cleared\n")
		}
		return m, nil
	}

	// Show help
	if IsKeyMatch(msg, "?") {
		m.State = StateHelpOverlay

		// Calculate overlay dimensions
		overlayWidth := int(float64(m.Width) * 0.7)
		overlayHeight := int(float64(m.Height) * 0.7)
		contentWidth := overlayWidth - 6    // 6 = 2*2 padding + 2 border
		viewportHeight := overlayHeight - 6 // Account for padding and borders

		// Set viewport dimensions
		m.HelpViewport.Width = contentWidth
		m.HelpViewport.Height = viewportHeight

		// Generate and set content
		content := m.KeyBindings.GenerateHelpContent(overlayWidth)
		m.HelpViewport.SetContent(content)
		m.HelpViewport.GotoTop()

		return m, nil
	}

	return m, nil
}

// handleDetailsOverlayKey handles key presses when in the details overlay state
func (m Model) handleDetailsOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for keys that close the details overlay
	if IsKeyMatch(msg, "esc/i") {
		m.State = StateNormal
	}
	return m, nil
}

// handleHelpOverlayKey handles key presses when in the help overlay state
func (m Model) handleHelpOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for keys that close the help overlay
	if IsKeyMatch(msg, "esc") || IsKeyMatch(msg, "?") {
		m.State = StateNormal
		return m, nil
	}

	// Handle navigation within help overlay
	if IsKeyMatch(msg, "up") || IsKeyMatch(msg, "k") {
		m.HelpViewport.LineUp(1)
	} else if IsKeyMatch(msg, "down") || IsKeyMatch(msg, "j") {
		m.HelpViewport.LineDown(1)
	} else if IsKeyMatch(msg, "pgup/pgdn") {
		if msg.String() == "pgup" {
			m.HelpViewport.HalfViewUp()
		} else {
			m.HelpViewport.HalfViewDown()
		}
	} else if IsKeyMatch(msg, "home/end") {
		if msg.String() == "home" {
			m.HelpViewport.GotoTop()
		} else {
			m.HelpViewport.GotoBottom()
		}
	}
	return m, nil
}
