package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"

	"github.com/Aj4x/tash/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("tash error: " + err.Error())
		os.Exit(1)
	}
}
