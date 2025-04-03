package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

func main() {

}

type model struct {
}

func (m model) Init() {}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return nil, nil
}

func (m model) View() string {
	return ""
}
