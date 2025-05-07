package main

import (
	"flag"
	"fmt"
	"github.com/Aj4x/tash/internal/msgbus"
	"github.com/Aj4x/tash/internal/task"
	"github.com/Aj4x/tash/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"runtime/debug"
)

func main() {
	// Parse command-line flags
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *versionFlag {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("tash: unable to read build info")
			os.Exit(1)
		}
		fmt.Printf("tash version %s\n", bi.Main.Version)
		os.Exit(0)
	}

	messageBus := msgbus.NewMessageBus[task.Message]()

	p := tea.NewProgram(ui.NewModel(messageBus), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("tash error: " + err.Error())
		os.Exit(1)
	}
}
